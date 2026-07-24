#!/bin/bash
# Comprehensive PWA & Security Validation Script

echo "========================================"
echo "  Syzhaa File - PWA Validation"
echo "========================================"
echo ""

PROJECT_DIR="/www/wwwroot/file.syzhaa.my.id"
FRONTEND_DIR="$PROJECT_DIR/frontend"
DOMAIN="https://file.syzhaa.my.id"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }

echo "▶ Step 1: Checking Files..."
cd $PROJECT_DIR

# Root files
for file in robots.txt sitemap.xml .htaccess; do
    if [ -f "$file" ]; then
        pass "$file exists"
    else
        fail "$file missing"
    fi
done

cd $FRONTEND_DIR

# Frontend files
for file in manifest.json sw.js offline.html; do
    if [ -f "$file" ]; then
        pass "$file exists"
    else
        fail "$file missing"
    fi
done

# Icons
ICON_COUNT=$(ls icon-*.png 2>/dev/null | wc -l)
if [ $ICON_COUNT -ge 10 ]; then
    pass "$ICON_COUNT PWA icons generated"
else
    warn "Only $ICON_COUNT icons found (expected 13)"
fi

for file in apple-touch-icon.png favicon.ico og-image.jpg; do
    if [ -f "$file" ]; then
        pass "$file exists"
    else
        fail "$file missing"
    fi
done

echo ""
echo "▶ Step 2: Validating manifest.json..."
if [ -f "manifest.json" ]; then
    if jq empty manifest.json 2>/dev/null; then
        pass "manifest.json is valid JSON"
    else
        fail "manifest.json has syntax errors"
    fi
    
    # Check required fields
    if jq -e '.name' manifest.json >/dev/null 2>&1; then
        NAME=$(jq -r '.name' manifest.json)
        pass "name: $NAME"
    else
        fail "name field missing"
    fi
    
    if jq -e '.icons | length' manifest.json >/dev/null 2>&1; then
        ICON_DEFS=$(jq '.icons | length' manifest.json)
        pass "icons defined: $ICON_DEFS"
    else
        warn "No icons defined in manifest"
    fi
fi

echo ""
echo "▶ Step 3: Validating index.html..."
cd $FRONTEND_DIR

# Check meta tags
if grep -q 'name="description"' index.html; then
    pass "SEO description exists"
else
    fail "SEO description missing"
fi

if grep -q 'property="og:' index.html; then
    OG_COUNT=$(grep -c 'property="og:' index.html)
    pass "Open Graph tags: $OG_COUNT"
else
    fail "Open Graph tags missing"
fi

if grep -q 'name="twitter:' index.html; then
    pass "Twitter Card tags exist"
else
    warn "Twitter Card tags missing"
fi

if grep -q 'rel="manifest"' index.html; then
    pass "manifest.json linked"
else
    fail "manifest.json not linked"
fi

if grep -q 'serviceWorker.register' index.html; then
    pass "Service Worker registered"
else
    fail "Service Worker not registered"
fi

if grep -q 'preconnect' index.html; then
    PRECONNECT_COUNT=$(grep -c 'preconnect' index.html)
    pass "Preconnect links: $PRECONNECT_COUNT"
else
    warn "No preconnect optimization"
fi

if grep -q 'csrf-token' index.html; then
    pass "CSRF token meta exists"
else
    warn "CSRF token meta missing"
fi

echo ""
echo "▶ Step 4: Testing Service Worker..."
if [ -f "sw.js" ]; then
    if grep -q "addEventListener.*install" sw.js; then
        pass "Install event handler exists"
    fi
    if grep -q "addEventListener.*fetch" sw.js; then
        pass "Fetch event handler exists"
    fi
    if grep -q "addEventListener.*activate" sw.js; then
        pass "Activate event handler exists"
    fi
fi

echo ""
echo "▶ Step 5: Checking Security Headers (.htaccess)..."
cd $PROJECT_DIR
if [ -f ".htaccess" ]; then
    if grep -q "X-Content-Type-Options" .htaccess; then
        pass "X-Content-Type-Options header"
    fi
    if grep -q "X-Frame-Options" .htaccess; then
        pass "X-Frame-Options header"
    fi
    if grep -q "Content-Security-Policy" .htaccess; then
        pass "Content-Security-Policy header"
    fi
    if grep -q "Strict-Transport-Security" .htaccess; then
        pass "HSTS header"
    fi
fi

echo ""
echo "▶ Step 6: File Sizes..."
cd $FRONTEND_DIR
echo "index.html: $(stat -c%s index.html 2>/dev/null || stat -f%z index.html) bytes"
echo "manifest.json: $(stat -c%s manifest.json 2>/dev/null || stat -f%z manifest.json) bytes"
echo "sw.js: $(stat -c%s sw.js 2>/dev/null || stat -f%z sw.js) bytes"
echo "og-image.jpg: $(du -h og-image.jpg 2>/dev/null | cut -f1)"

echo ""
echo "========================================"
echo "  Validation Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo "1. Test live: curl -I $DOMAIN"
echo "2. Lighthouse: Chrome DevTools > Lighthouse"
echo "3. PWA Test: $DOMAIN in mobile browser"
echo "4. Security: https://securityheaders.com/?q=$DOMAIN"
echo ""
echo "Restart app: pm2 restart ecosystem.config.js"
echo ""

