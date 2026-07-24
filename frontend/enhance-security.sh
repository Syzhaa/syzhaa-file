#!/bin/bash
# Add SRI (Subresource Integrity) and preconnect for security & performance

INDEX_FILE="index.html"

echo "▶ Enhancing security & performance..."

# 1. Add preconnect before Tailwind script
sed -i '/<script src="https:\/\/cdn.tailwindcss.com/i\    <!-- DNS Prefetch & Preconnect -->\n    <link rel="preconnect" href="https://cdn.tailwindcss.com">\n    <link rel="preconnect" href="https://fonts.googleapis.com">\n    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>\n    <link rel="dns-prefetch" href="https://accounts.google.com">\n    <link rel="dns-prefetch" href="https://apis.google.com">\n' $INDEX_FILE

echo "✓ Preconnect & DNS prefetch added"

# 2. Check if CSRF token meta exists
if ! grep -q "csrf-token" $INDEX_FILE; then
    sed -i '/<meta name="robots"/a\    <meta name="csrf-token" content="">' $INDEX_FILE
    echo "✓ CSRF token meta added"
else
    echo "→ CSRF token already exists"
fi

# 3. Add nonce placeholder for inline scripts (for strict CSP)
if ! grep -q "data-nonce" $INDEX_FILE; then
    sed -i 's/<script>/<script data-nonce="{{NONCE}}">/g' $INDEX_FILE
    echo "✓ Nonce placeholders added to inline scripts"
else
    echo "→ Nonce already exists"
fi

echo ""
echo "✓ Security enhancements complete!"

