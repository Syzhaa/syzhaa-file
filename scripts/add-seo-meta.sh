#!/bin/bash
# Add SEO Meta Tags to index.html

INDEX_FILE="index.html"

# Find line number after <title>
LINE_NUM=$(grep -n "<title>" $INDEX_FILE | cut -d: -f1)
AFTER_LINE=$((LINE_NUM + 1))

# Create SEO meta tags
cat > /tmp/seo-meta.txt << 'EOFMETA'
    
    <!-- SEO Meta Tags -->
    <meta name="description" content="Syzhaa File adalah platform berbagi file modern dengan enkripsi end-to-end. Buat ruang file, bagikan dengan mudah, dan kelola file Anda dengan aman.">
    <meta name="keywords" content="berbagi file, file sharing, transfer file, enkripsi file, upload file, syzhaa file, file storage">
    <meta name="author" content="Syzhaa">
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="https://file.syzhaa.my.id/">
    
    <!-- Open Graph Meta Tags -->
    <meta property="og:type" content="website">
    <meta property="og:title" content="Syzhaa File - Platform Berbagi File Profesional">
    <meta property="og:description" content="Platform berbagi file modern dengan enkripsi end-to-end. Buat ruang file, bagikan dengan mudah, dan kelola file Anda dengan aman.">
    <meta property="og:url" content="https://file.syzhaa.my.id/">
    <meta property="og:image" content="https://file.syzhaa.my.id/og-image.jpg">
    <meta property="og:image:width" content="1200">
    <meta property="og:image:height" content="630">
    <meta property="og:site_name" content="Syzhaa File">
    <meta property="og:locale" content="id_ID">
    
    <!-- Twitter Card Meta Tags -->
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="Syzhaa File - Platform Berbagi File Profesional">
    <meta name="twitter:description" content="Platform berbagi file modern dengan enkripsi end-to-end. Buat ruang file, bagikan dengan mudah.">
    <meta name="twitter:image" content="https://file.syzhaa.my.id/og-image.jpg">
    <meta name="twitter:site" content="@syzhaa">
    <meta name="twitter:creator" content="@syzhaa">
    
    <!-- Favicon Links -->
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
    <link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png">
    <link rel="shortcut icon" href="/favicon.ico">
EOFMETA

# Insert SEO meta tags after title
sed -i "${AFTER_LINE}r /tmp/seo-meta.txt" $INDEX_FILE

echo "✓ SEO meta tags added to index.html"

# Also create favicon-32x32.png and favicon-16x16.png (symlinks)
ln -sf icon-32x32.png favicon-32x32.png
ln -sf icon-16x16.png favicon-16x16.png

echo "✓ Favicon symlinks created"

rm /tmp/seo-meta.txt

