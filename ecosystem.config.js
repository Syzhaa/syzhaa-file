module.exports = {
  apps: [{
    name: 'ambilfile',
    script: './file-server',
    cwd: '/www/wwwroot/ambilfile',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env: {
      PORT: '4006',
      BASE_URL: 'http://ambilfile.web.id',
      ADMIN_EMAIL: 'admin@ambilfile.web.id',
      ADMIN_EMAILS: 'syzhaadigital@gmail.com,admin@ambilfile.web.id',
      ADMIN_NAME: 'Administrator',
      ADMIN_PASSWORD: 'admin',
      DB_PATH: './data/files.db',
      NODE_ENV: 'production'
    }
  }]
};
