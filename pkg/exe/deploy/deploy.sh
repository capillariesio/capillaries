# Create all instances in one shot (54 s)
go run deploy.go create_instances bastion,cass01,cass02,rabbitmq,prometheus,daemon01

# Create sftp user
go run deploy.go create_instance_users bastion

# Allow these hosts to connect to data via sftp
go run deploy.go copy_private_keys bastion,daemon01

# Volumes: used only by bastion
go run deploy.go create_volumes
# Requires sftpuser
go run deploy.go attach_volumes bastion

# Upload all files in one shot (0.5 min)
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui,up_test_in,up_test_out,up_test_cfg

# Setup all services except daemons (2 min)
go run deploy.go setup_services bastion,cass01,cass02,prometheus,rabbitmq

# Start capidaemons (30 s)
go run deploy.go setup_services daemon01





go run deploy.go create_instances bastion,daemon01

go run deploy.go create_instance_users bastion
go run deploy.go copy_private_keys bastion

go run deploy.go create_volumes
go run deploy.go attach_volumes bastion

go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui,up_test_in,up_test_cfg

#go run deploy.go setup_services daemon01
