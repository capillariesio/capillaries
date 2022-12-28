go run deploy.go create_instances bastion,rabbitmq,daemon01
go run deploy.go setup_services bastion,rabbitmq
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary
go run deploy.go setup_services daemon01

# go run deploy.go delete_instances bastion,rabbitmq,daemon01
# go run deploy.go download_files down_capi_logs