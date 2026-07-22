module.exports = {
  apps: [{
    name: 'syzhaa-file-go',
    script: './file-server',
    cwd: '/www/wwwroot/file-go',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env_file: '.env'
  }]
};
