module.exports = {
  apps: [{
    name: 'syzhaa-file',
    script: './file-server',
    cwd: '/www/wwwroot/file.syzhaa.my.id/backend',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env: {
      GOOGLE_CLIENT_ID: '974513868394-cil69on4itbnt4dpjtngojjrqrkm5k9j.apps.googleusercontent.com',
      GOOGLE_CLIENT_SECRET: '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',
      GOOGLE_REDIRECT_URL: 'https://file.syzhaa.my.id/auth/google/callback',
      BASE_URL: 'https://file.syzhaa.my.id',
      ADMIN_EMAILS: 'syzhaadigital@gmail.com'
    }
  }]
};
