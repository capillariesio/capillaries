# Create all instances in one shot (2 min)
go run deploy.go create_instances bastion,cass01,cass02,cass03,rabbitmq,prometheus,daemon01

# Volumes: used only by bastion (upload/download,webapi) and capidaemons (1 min)
go run deploy.go create_volumes
go run deploy.go attach_volumes bastion,daemon01

# Upload all files in one shot (2 min)
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui

# Start all services except daemons (30 s)
go run deploy.go setup_services bastion,cass01,cass02,cass03,prometheus,rabbitmq

# Start capidaemons (30 s)
go run deploy.go setup_services daemon01
