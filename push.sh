#!/usr/bin/env bash
# Push syzhaa-file otomatis. Pake gh credential (udah kepasang).
# Pakai: bash ~/workspace/syzhaa-file/push.sh
set -e
cd ~/workspace/syzhaa-file
git add -A
git commit -m "auto: $(date +%F\ %H:%M)" || echo "nothing to commit"
git push origin main
echo "OK pushed: $(git log --oneline -1)"
