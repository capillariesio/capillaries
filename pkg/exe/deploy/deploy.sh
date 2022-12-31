# Create all instances in one shot (54 s)
go run deploy.go create_instances bastion,cass01,cass02,rabbitmq,prometheus,daemon01

# Volumes: used only by bastion (upload/download,webapi) and capidaemons (0.5 min)
go run deploy.go create_volumes
go run deploy.go attach_volumes bastion,daemon01

# Upload all files in one shot (0.5 min)
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui,up_test_in,up_test_scripts

# Setup all services except daemons (2 min)
go run deploy.go setup_services bastion,cass01,cass02,prometheus,rabbitmq

# Start capidaemons (30 s)
go run deploy.go setup_services daemon01
