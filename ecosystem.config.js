module.exports = {
  apps: [{
    name: 'syzhaa-file',
    script: './file-server',
    cwd: '/www/wwwroot/file.syzhaa.my.id/backend',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env_file: '.env',
    env: {
      ADMIN_EMAILS: 'syzhaadigital@gmail.com'
    }
  }]
};
