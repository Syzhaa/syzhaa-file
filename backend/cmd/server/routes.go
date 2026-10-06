package main

import (
	"github.com/syzhaa/file-server/internal/admin"
	"github.com/syzhaa/file-server/internal/apikeys"
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/files"
	"github.com/syzhaa/file-server/internal/middleware"
	"github.com/syzhaa/file-server/internal/rooms"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.CorsMiddleware)
	r.Use(middleware.SecurityHeadersMiddleware)
	r.Use(middleware.CsrfOriginMiddleware)
	r.Use(middleware.RateLimitMiddleware(middleware.GeneralLimiter))

	// Public routes (existing)
	r.HandleFunc("/api/stats", getPublicStatsHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/room/create", rooms.CreateRoomHandler).Methods("POST", "OPTIONS")
	r.Handle("/api/room/pin", middleware.RateLimitMiddleware(middleware.PinLimiter)(http.HandlerFunc(rooms.AccessRoomByPinHandler))).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/{id}", rooms.GetRoomInfoHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/room/{id}", rooms.HandleDeleteRoom).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/room/{id}/settings", rooms.UpdateRoomSettingsHandler).Methods("PUT", "OPTIONS")
	r.Handle("/api/upload/{roomId}", middleware.RateLimitMiddleware(middleware.UploadLimiter)(http.HandlerFunc(files.UploadChunkHandler))).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/upload/{roomId}/chunks", files.UploadChunkStatusHandler).Methods("GET")
	r.HandleFunc("/api/file/{id}", files.DeleteFileHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/folder/create/{roomId}", files.CreateFolderHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/folders/{roomId}", files.ListFoldersHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/folder/{id}", files.DeleteFolderHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/d/{id}", files.DownloadFileHandler).Methods("GET")

	// Admin email/password auth routes
	r.Handle("/auth/login", middleware.RateLimitMiddleware(middleware.LoginLimiter)(http.HandlerFunc(auth.HandleAdminLogin))).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/logout", auth.HandleAdminLogout).Methods("POST", "OPTIONS")

	// Admin routes (require session)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(requireAdminSession)
	adminRouter.HandleFunc("/me", auth.HandleAdminMe).Methods("GET")
	adminRouter.HandleFunc("/stats", admin.HandleAdminStats).Methods("GET")
	adminRouter.HandleFunc("/api-keys", apikeys.HandleListAPIKeys).Methods("GET")
	adminRouter.HandleFunc("/api-keys", apikeys.HandleCreateAPIKey).Methods("POST")
	adminRouter.HandleFunc("/api-keys/{id}", apikeys.HandleDeleteAPIKey).Methods("DELETE")
	adminRouter.HandleFunc("/api-keys/{id}/toggle", apikeys.HandleToggleAPIKey).Methods("POST")

	// Admin user management routes
	adminRouter.HandleFunc("/users", admin.HandleAdminListUsers).Methods("GET")
	adminRouter.HandleFunc("/rooms", admin.HandleAdminListRooms).Methods("GET")
	adminRouter.HandleFunc("/users/{id}/approve", admin.HandleAdminApproveUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/reject", admin.HandleAdminRejectUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/suspend", admin.HandleAdminSuspendUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/quotas", admin.HandleAdminUpdateUserQuotas).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/api-approve", admin.HandleAdminApproveUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/api-revoke", admin.HandleAdminRevokeUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/stats", admin.HandleAdminGetUserStats).Methods("GET")
	adminRouter.HandleFunc("/settings", admin.HandleAdminSystemSettings).Methods("GET", "POST")
	adminRouter.HandleFunc("/account", auth.HandleAdminUpdateAccount).Methods("POST", "PUT")

	// auth.User routes (require user session)
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(auth.RequireUserSession)
	userRouter.HandleFunc("/me", auth.HandleUserMe).Methods("GET")
	userRouter.HandleFunc("/rooms", auth.HandleUserRooms).Methods("GET")
	userRouter.HandleFunc("/api-keys", apikeys.HandleUserListAPIKeys).Methods("GET")
	userRouter.HandleFunc("/api-keys", auth.HandleUserCreateAPIKey).Methods("POST")
	userRouter.HandleFunc("/api-keys/request", auth.HandleUserRequestAPIAccess).Methods("POST")
	userRouter.HandleFunc("/api-keys/{id}", apikeys.HandleUserDeleteAPIKey).Methods("DELETE")

	// auth.User auth (public)
	r.Handle("/auth/user/register", middleware.RateLimitMiddleware(middleware.LoginLimiter)(http.HandlerFunc(auth.HandleUserRegister))).Methods("POST", "OPTIONS")
	r.Handle("/auth/user/login", middleware.RateLimitMiddleware(middleware.LoginLimiter)(http.HandlerFunc(auth.HandleUserLogin))).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/logout", auth.HandleUserLogout).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/google", auth.HandleUserGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/user/google/callback", auth.HandleUserGoogleCallback).Methods("GET")

	// API v1 routes (require API key)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(requireAPIKey)
	apiRouter.HandleFunc("/room/create", admin.HandleAPICreateRoom).Methods("POST")
	apiRouter.HandleFunc("/room/{id}/link", admin.HandleAPIGetRoomLink).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/files", admin.HandleAPIGetRoomFiles).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/download-all", admin.HandleAPIDownloadAll).Methods("GET")

	// Clean URLs: /user -> user dashboard, /admin -> admin dashboard
	r.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/user/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/user/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")
	r.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)
	}).Methods("GET")

	r.PathPrefix("/").Handler(middleware.StaticCacheMiddleware(middleware.CleanURLMiddleware(http.FileServer(http.Dir("./frontend")))))

	log.Printf("🚀 AmbilFile Server (Go) running on port %d", Port)
	return r
}
