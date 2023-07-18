{
  // Variables to play with
  local cassandra_node_flavor = "x",
  local cassandra_total_nodes = 4,
  local daemon_total_instances = 2,
  local DEFAULT_DAEMON_THREAD_POOL_SIZE = '5',
  local DEFAULT_DAEMON_DB_WRITERS = '5',

  // Basics
  local deployment_name = 'sampledeployment003',  // Can be any combination of alphanumeric characters. Make it unique.
  local default_root_key_name = deployment_name + '-root-key',  // This should match the name of the keypair you already created in Openstack

  // Network
  local external_gateway_network_name = 'Ext-Net',  // This is what external network is called for this cloud provider, yours may be different
  local subnet_cidr = '10.5.0.0/24',  // Your choice
  local subnet_allocation_pool = 'start=10.5.0.240,end=10.5.0.254',  // We use fixed ip addresses in the .0.2-.0.239 range, the rest is potentially available

  // Internal IPs
  local internal_bastion_ip = '10.5.0.10',
  local prometheus_ip = '10.5.0.4',
  local rabbitmq_ip = '10.5.0.5',
  local daemon_ips = 
    if daemon_total_instances == 2 then ['10.5.0.101', '10.5.0.102']
    else if daemon_total_instances == 4 then ['10.5.0.101', '10.5.0.102', '10.5.0.103', '10.5.0.104']
    else if daemon_total_instances == 8 then ['10.5.0.101', '10.5.0.102', '10.5.0.103', '10.5.0.104', '10.5.0.105', '10.5.0.106', '10.5.0.107', '10.5.0.108']
    else [],
  local cassandra_ips = 
    if cassandra_total_nodes == 4 then ['10.5.0.11', '10.5.0.12', '10.5.0.13', '10.5.0.14']
    else if cassandra_total_nodes == 8 then ['10.5.0.11', '10.5.0.12', '10.5.0.13', '10.5.0.14', '10.5.0.15', '10.5.0.16', '10.5.0.17', '10.5.0.18']
    else if cassandra_total_nodes == 16 then ['10.5.0.11', '10.5.0.12', '10.5.0.13', '10.5.0.14', '10.5.0.15', '10.5.0.16', '10.5.0.17', '10.5.0.18', '10.5.0.19', '10.5.0.20', '10.5.0.21', '10.5.0.22', '10.5.0.23', '10.5.0.24', '10.5.0.25', '10.5.0.26']
    else [],

  // Cassandra-specific
  local cassandra_tokens = // Initial tokens to speedup bootstrapping
    if cassandra_total_nodes == 4 then ['-9223372036854775808', '-4611686018427387904', '0', '4611686018427387904']
    else if cassandra_total_nodes == 8 then ['-9223372036854775808', '-6917529027641081856', '-4611686018427387904', '-2305843009213693952', '0', '2305843009213693952', '4611686018427387904', '6917529027641081856']
    else if cassandra_total_nodes == 16 then ['-9223372036854775808','-8070450532247928832','-6917529027641081856','-5764607523034234880','-4611686018427387904','-3458764513820540928','-2305843009213693952','-1152921504606846976','0','1152921504606846976','2305843009213693952','3458764513820540928','4611686018427387904','5764607523034234880','6917529027641081856','8070450532247928832']
    else [],
  local cassandra_seeds = std.format('%s,%s', [cassandra_ips[0], cassandra_ips[1]]),  // Used by cassandra nodes
  local cassandra_hosts = "'[\"" + std.join('","', cassandra_ips) + "\"]'",  // Used by daemons "'[\"10.5.0.11\",\"10.5.0.12\",\"10.5.0.13\",\"10.5.0.14\",\"10.5.0.15\",\"10.5.0.16\",\"10.5.0.17\",\"10.5.0.18\"]'",
  
  // Instance details
  local default_availability_zone = 'nova',  // Specified when volume/instance is created
  local instance_image_name = 'Ubuntu 23.04',
  local instance_flavor_rabbitmq = 'b2-7',
  local instance_flavor_prometheus = 'b2-7',
  local instance_flavor_bastion =
    if cassandra_node_flavor == "x" then 'b2-7'
    else if cassandra_node_flavor == "2x" then ''
    else if cassandra_node_flavor == "4x" then ''
    else "unknown",
  local instance_flavor_cassandra =
    if cassandra_node_flavor == "x" then 'b2-7'
    else if cassandra_node_flavor == "2x" then ''
    else if cassandra_node_flavor == "4x" then ''
    else "unknown",
  local instance_flavor_daemon =
    if cassandra_node_flavor == "x" then 'b2-7'
    else if cassandra_node_flavor == "2x" then ''
    else if cassandra_node_flavor == "4x" then ''
    else "unknown",
  
  // Artifacts
  local buildLinuxAmd64Dir = '../../build/linux/amd64',
  local pkgExeDir = '../../pkg/exe',
  
  // Keys
  local sftp_config_public_key_path = '~/.ssh/sampledeployment003_sftp.pub',
  local sftp_config_private_key_path = '~/.ssh/sampledeployment003_sftp',
  local ssh_config_private_key_path = '~/.ssh/sampledeployment003_rsa',
  
  // Prometheus versions
  local prometheus_node_exporter_version = '1.6.0',
  local prometheus_server_version = '2.45.0',
  local prometheus_cassandra_exporter_version = '0.9.12',

  // Used by Prometheus "\\'localhost:9100\\',\\'10.5.0.10:9100\\',\\'10.5.0.5:9100\\',\\'10.5.0.11:9100\\'...",
  local prometheus_targets = std.format("\\'localhost:9100\\',\\'%s:9100\\',\\'%s:9100\\',", [internal_bastion_ip, rabbitmq_ip]) +
                             "\\'" + std.join(":9100\\',\\'", cassandra_ips) + ":9100\\'," +
                             "\\'" + std.join(":9500\\',\\'", cassandra_ips) + ":9500\\'," + // Cassandra exporter
                             "\\'" + std.join(":9100\\',\\'", daemon_ips) + ":9100\\'",

  // Full list of env variables expected by capideploy working with this project
  env_variables_used: [
    // Used in this config
    'CAPIDEPLOY_SSH_USER',
    'CAPIDEPLOY_SSH_PRIVATE_KEY_PASS',
    'CAPIDEPLOY_SFTP_USER',
    'CAPIDEPLOY_RABBITMQ_ADMIN_NAME',
    'CAPIDEPLOY_RABBITMQ_ADMIN_PASS',
    'CAPIDEPLOY_RABBITMQ_USER_NAME',
    'CAPIDEPLOY_RABBITMQ_USER_PASS',
    // Used in by Capideploy Openstack calls
    'OS_AUTH_URL',
    'OS_IDENTITY_API_VERSION',
    'OS_INTERFACE',
    'OS_REGION_NAME',
    'OS_PASSWORD',
    'OS_PROJECT_DOMAIN_ID',
    'OS_PROJECT_ID',
    'OS_PROJECT_NAME',
    'OS_USERNAME',
    'OS_USER_DOMAIN_NAME',
  ],
  ssh_config: {
    external_ip_address: '',
    port: 22,
    user: '{CAPIDEPLOY_SSH_USER}',
    private_key_path: ssh_config_private_key_path,
    private_key_password: '{CAPIDEPLOY_SSH_PRIVATE_KEY_PASS}',
  },
  timeouts: {
    openstack_cmd: 60,
    openstack_instance_creation: 240,
    attach_volume: 60,
  },

  // It's unlikely that you need to change anything below this line

  artifacts: {
    env: {
      DIR_BUILD_LINUX_AMD64: '../../' + buildLinuxAmd64Dir,
      DIR_PKG_EXE: '../../' + pkgExeDir,
      DIR_CODE_PARQUET: '../../../code/parquet',
    },
    cmd: [
      'sh/local/build_binaries.sh',
      'sh/local/build_webui.sh',
      'sh/local/prepare_demo_data.sh',
    ],
  },

  network: {
    name: deployment_name + '_network',
    subnet: {
      name: deployment_name + '_subnet',
      cidr: subnet_cidr,
      allocation_pool: subnet_allocation_pool,
    },
    router: {
      name: deployment_name + '_router',
      external_gateway_network_name: external_gateway_network_name,
    },
  },
  security_groups: {
    bastion: {
      name: deployment_name + '_bastion_security_group',
      rules: [
        {
          desc: 'SSH',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: '0.0.0.0/0',
          port: 22,
          direction: 'ingress',
        },
        {
          desc: 'NFS PortMapper TCP',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 111,
          direction: 'ingress',
        },
        {
          desc: 'NFS PortMapper UDP',
          protocol: 'udp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 111,
          direction: 'ingress',
        },
        {
          desc: 'NFS Server TCP',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 2049,
          direction: 'ingress',
        },
        {
          desc: 'NFS Server UDP',
          protocol: 'udp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 2049,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus UI reverse proxy',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: '0.0.0.0/0',
          port: 9090,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus node exporter',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 9100,
          direction: 'ingress',
        },
        {
          desc: 'RabbitMQ UI reverse proxy',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: '0.0.0.0/0',
          port: 15672,
          direction: 'ingress',
        },
        {
          desc: 'rsyslog receiver',
          protocol: 'udp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 514,
          direction: 'ingress',
        },
        {
          desc: 'Capillaries webapi',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: '0.0.0.0/0',
          port: 6543,
          direction: 'ingress',
        },
        {
          desc: 'Capillaries UI nginx',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: '0.0.0.0/0',
          port: 80,
          direction: 'ingress',
        },
      ],
    },
    internal: {
      name: deployment_name + '_internal_security_group',
      rules: [
        {
          desc: 'SSH',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 22,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus UI internal',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 9090,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus node exporter',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 9100,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra Prometheus node exporter',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 9500,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra JMX',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 7199,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra cluster comm',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 7000,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra API',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 9042,
          direction: 'ingress',
        },
        {
          desc: 'RabbitMQ API',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 5672,
          direction: 'ingress',
        },
        {
          desc: 'RabbitMQ UI',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.subnet.cidr,
          port: 15672,
          direction: 'ingress',
        },
      ],
    },
  },
  file_groups_up: {
    up_all_cfg: {
      src: '/tmp/capi_cfg',
      dst: '/mnt/capi_cfg',
      dir_permissions: 777,
      file_permissions: 666,
      owner: '{CAPIDEPLOY_SSH_USER}',
      after: {
        env: {
          LOCAL_CFG_LOCATION: '/mnt/capi_cfg',
          MOUNT_POINT_CFG: '/mnt/capi_cfg', // If SFTP used: 'sftp://{CAPIDEPLOY_SFTP_USER}@' + internal_bastion_ip + '/mnt/capi_cfg',
          MOUNT_POINT_IN: '/mnt/capi_in',
          MOUNT_POINT_OUT: '/mnt/capi_out',
        },
        cmd: [
          'sh/capiscripts/adjust_cfg_in_out.sh',
        ],
      },
    },
    up_capiparquet_binary: {
      src: buildLinuxAmd64Dir + '/capiparquet.gz',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {
        env: {
          CAPI_BINARY: '/home/' + $.ssh_config.user + '/bin/capiparquet',
        },
        cmd: [
          'sh/capiscripts/unpack_capi_binary.sh',
        ],
      },
    },
    up_daemon_binary: {
      src: buildLinuxAmd64Dir + '/capidaemon.gz',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {
        env: {
          CAPI_BINARY: '/home/' + $.ssh_config.user + '/bin/capidaemon',
        },
        cmd: [
          'sh/capiscripts/unpack_capi_binary.sh',
        ],
      },
    },
    up_daemon_env_config: {
      src: pkgExeDir + '/daemon/capidaemon.json',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {},
    },
    up_diff_scripts: {
      src: './diff',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 744,
      after: {},
    },
    up_lookup_bigtest_in: {
      src: '/tmp/capi_in/lookup_bigtest/all.tgz',
      dst: '/mnt/capi_in/lookup_bigtest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {
        env: {
          OWNER_USER: $.ssh_config.user,
        },
        cmd: [
          'sh/capiscripts/unpack_lookup_big_in.sh',
        ],
      },
    },
    up_lookup_bigtest_out: {
      src: '/tmp/capi_out/lookup_bigtest/all.tgz',
      dst: '/mnt/capi_out/lookup_bigtest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {
        env: {
          OWNER_USER: $.ssh_config.user,
        },
        cmd: [
          'sh/capiscripts/unpack_lookup_big_out.sh',
        ],
      },
    },
    up_lookup_quicktest_in: {
      src: '/tmp/capi_in/lookup_quicktest',
      dst: '/mnt/capi_in/lookup_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_lookup_quicktest_out: {
      src: '/tmp/capi_out/lookup_quicktest',
      dst: '/mnt/capi_out/lookup_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_portfolio_quicktest_in: {
      src: '/tmp/capi_in/portfolio_quicktest',
      dst: '/mnt/capi_in/portfolio_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_portfolio_quicktest_out: {
      src: '/tmp/capi_out/portfolio_quicktest',
      dst: '/mnt/capi_out/portfolio_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_py_calc_quicktest_in: {
      src: '/tmp/capi_in/py_calc_quicktest',
      dst: '/mnt/capi_in/py_calc_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_py_calc_quicktest_out: {
      src: '/tmp/capi_out/py_calc_quicktest',
      dst: '/mnt/capi_out/py_calc_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_tag_and_denormalize_quicktest_in: {
      src: '/tmp/capi_in/tag_and_denormalize_quicktest',
      dst: '/mnt/capi_in/tag_and_denormalize_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_tag_and_denormalize_quicktest_out: {
      src: '/tmp/capi_out/tag_and_denormalize_quicktest',
      dst: '/mnt/capi_out/tag_and_denormalize_quicktest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {},
    },
    up_toolbelt_binary: {
      src: buildLinuxAmd64Dir + '/capitoolbelt.gz',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {
        env: {
          CAPI_BINARY: '/home/' + $.ssh_config.user + '/bin/capitoolbelt',
        },
        cmd: [
          'sh/capiscripts/unpack_capi_binary.sh',
        ],
      },
    },
    up_toolbelt_env_config: {
      src: pkgExeDir + '/toolbelt/capitoolbelt.json',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {},
    },
    up_ui: {
      src: '../../ui/public',
      dst: '/home/' + $.ssh_config.user + '/ui',
      dir_permissions: 755,
      file_permissions: 644,
      after: {},
    },
    up_webapi_binary: {
      src: buildLinuxAmd64Dir + '/capiwebapi.gz',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {
        env: {
          CAPI_BINARY: '/home/' + $.ssh_config.user + '/bin/capiwebapi',
        },
        cmd: [
          'sh/capiscripts/unpack_capi_binary.sh',
        ],
      },
    },
    up_webapi_env_config: {
      src: pkgExeDir + '/webapi/capiwebapi.json',
      dst: '/home/' + $.ssh_config.user + '/bin',
      dir_permissions: 744,
      file_permissions: 644,
      after: {},
    },
  },
  file_groups_down: {
    down_capi_logs: {
      src: '/var/log/capidaemon/',
      dst: './tmp/capi_logs',
    },
    down_capi_out: {
      src: '/mnt/capi_out',
      dst: './tmp/capi_out',
    },
  },

  // Only alphanumeric characters allowed in instance names! No underscores, no dashes, no dots, no spaces - nada.

  local bastion_instance = {
    bastion: {
      host_name: deployment_name + '-bastion',
      security_group: 'bastion',
      root_key_name: default_root_key_name,
      ip_address: internal_bastion_ip,
      uses_ssh_config_external_ip_address: true,
      flavor: instance_flavor_bastion,
      image: instance_image_name,
      availability_zone: default_availability_zone,
      volumes: {
        cfg: {
          name: deployment_name + '_cfg',
          availability_zone: default_availability_zone,
          mount_point: '/mnt/capi_cfg',
          size: 1,
          type: 'classic',
          permissions: 777,
          owner: $.ssh_config.user, // If SFTP used: "{CAPIDEPLOY_SFTP_USER}"
        },
        'in': {
          name: deployment_name + '_in',
          availability_zone: default_availability_zone,
          mount_point: '/mnt/capi_in',
          size: 1,
          type: 'classic',
          permissions: 777,
          owner: $.ssh_config.user,
        },
        out: {
          name: deployment_name + '_out',
          availability_zone: default_availability_zone,
          mount_point: '/mnt/capi_out',
          size: 1,
          type: 'classic',
          permissions: 777,
          owner: $.ssh_config.user,
        },
      },
      users: [
        {
          name: '{CAPIDEPLOY_SFTP_USER}',
          public_key_path: sftp_config_public_key_path,
        },
      ],
      private_keys: [
        {
          name: '{CAPIDEPLOY_SFTP_USER}',
          private_key_path: sftp_config_private_key_path,
        },
      ],
      service: {
        env: {
          AMQP_URL: 'amqp://{CAPIDEPLOY_RABBITMQ_USER_NAME}:{CAPIDEPLOY_RABBITMQ_USER_PASS}@' + rabbitmq_ip + '/',
          CASSANDRA_HOSTS: cassandra_hosts,
          NFS_DIRS: '/mnt/capi_cfg,/mnt/capi_in,/mnt/capi_out',
          PROMETHEUS_IP: prometheus_ip,
          PROMETHEUS_NODE_EXPORTER_VERSION: prometheus_node_exporter_version,
          RABBITMQ_IP: rabbitmq_ip,
          SFTP_USER: '{CAPIDEPLOY_SFTP_USER}',
          SSH_USER: $.ssh_config.user,
          SUBNET_CIDR: $.network.subnet.cidr,
          EXTERNAL_IP_ADDRESS: '{EXTERNAL_IP_ADDRESS}',  // internal: capideploy populates it from ssh_config.external_ip_address after loading project file; used by webui and webapi config.sh
          WEBAPI_PORT: '6543',
        },
        cmd: {
          install: [
            'sh/common/replace_nameserver.sh',
            'sh/common/increase_ssh_connection_limit.sh',
            'sh/prometheus/install_node_exporter.sh',
            'sh/nfs/install_server.sh',
            'sh/nginx/install.sh',
          ],
          config: [
            'sh/prometheus/config_node_exporter.sh',
            'sh/rsyslog/config_capidaemon_log_receiver.sh',
            'sh/logrotate/config_capidaemon_logrotate.sh',
            'sh/toolbelt/config.sh',
            'sh/webapi/config.sh',
            'sh/ui/config.sh',
            'sh/nginx/config_ui.sh',
            'sh/nfs/config_server.sh',
            'sh/nginx/config_prometheus_reverse_proxy.sh',
            'sh/nginx/config_rabbitmq_reverse_proxy.sh',
          ],
          start: [
            'sh/webapi/start.sh',
            'sh/nginx/start.sh',
          ],
          stop: [
            'sh/webapi/stop.sh',
            'sh/nginx/stop.sh',
          ],
        },
      },
      applicable_file_groups: [
        'up_all_cfg',
        'up_lookup_bigtest_in',
        'up_lookup_bigtest_out',
        'up_lookup_quicktest_in',
        'up_lookup_quicktest_out',
        'up_tag_and_denormalize_quicktest_in',
        'up_tag_and_denormalize_quicktest_out',
        'up_py_calc_quicktest_in',
        'up_py_calc_quicktest_out',
        'up_portfolio_quicktest_in',
        'up_portfolio_quicktest_out',
        'up_webapi_binary',
        'up_webapi_env_config',
        'up_toolbelt_binary',
        'up_toolbelt_env_config',
        'up_capiparquet_binary',
        'up_ui',
        'up_diff_scripts',
        'down_capi_out',
        'down_capi_logs',
      ],
    },
  },

  local rabbitmq_instance = {
    rabbitmq: {
      host_name: deployment_name + '-rabbitmq',
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: rabbitmq_ip,
      flavor: instance_flavor_rabbitmq,
      image: instance_image_name,
      availability_zone: default_availability_zone,
      service: {
        env: {
          PROMETHEUS_NODE_EXPORTER_VERSION: prometheus_node_exporter_version,
          RABBITMQ_ADMIN_NAME: '{CAPIDEPLOY_RABBITMQ_ADMIN_NAME}',
          RABBITMQ_ADMIN_PASS: '{CAPIDEPLOY_RABBITMQ_ADMIN_PASS}',
          RABBITMQ_USER_NAME: '{CAPIDEPLOY_RABBITMQ_USER_NAME}',
          RABBITMQ_USER_PASS: '{CAPIDEPLOY_RABBITMQ_USER_PASS}',
        },
        cmd: {
          install: [
            'sh/common/replace_nameserver.sh',
            'sh/prometheus/install_node_exporter.sh',
            'sh/rabbitmq/install.sh',
          ],
          config: [
            'sh/prometheus/config_node_exporter.sh',
            'sh/rabbitmq/config.sh',
          ],
          start: [
            'sh/rabbitmq/start.sh',
          ],
          stop: [
            'sh/rabbitmq/stop.sh',
          ],
        },
      },
    },
  },

  local prometheus_instance = {
    prometheus: {
      host_name: deployment_name + '-prometheus',
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: prometheus_ip,
      flavor: instance_flavor_prometheus,
      image: instance_image_name,
      availability_zone: default_availability_zone,
      service: {
        env: {
          PROMETHEUS_NODE_EXPORTER_VERSION: prometheus_node_exporter_version,
          PROMETHEUS_TARGETS: prometheus_targets,
          PROMETHEUS_VERSION: prometheus_server_version,
        },
        cmd: {
          install: [
            'sh/common/replace_nameserver.sh',
            'sh/prometheus/install_server.sh',
            'sh/prometheus/install_node_exporter.sh',
          ],
          config: [
            'sh/prometheus/config_server.sh',
            'sh/prometheus/config_node_exporter.sh',
          ],
          start: [
            'sh/prometheus/start_server.sh',
          ],
          stop: [
            'sh/prometheus/stop_server.sh',
          ],
        },
      },
    },
  },

  local cass_instances = {
    [e.nickname]: {
      host_name: e.host_name,
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: e.ip_address,
      flavor: instance_flavor_cassandra,
      image: instance_image_name,
      availability_zone: default_availability_zone,
      service: {
        env: {
          CASSANDRA_IP: e.ip_address,
          CASSANDRA_SEEDS: cassandra_seeds,
          INITIAL_TOKEN: e.token,
          PROMETHEUS_NODE_EXPORTER_VERSION: prometheus_node_exporter_version,
          PROMETHEUS_CASSANDRA_EXPORTER_VERSION: prometheus_cassandra_exporter_version,
        },
        cmd: {
          install: [
            'sh/common/replace_nameserver.sh',
            'sh/prometheus/install_node_exporter.sh',
            'sh/cassandra/install.sh',
          ],
          config: [
            'sh/prometheus/config_node_exporter.sh',
            'sh/cassandra/config.sh',
          ],
          start: [
            'sh/cassandra/start.sh',
          ],
          stop: [
            'sh/cassandra/stop.sh',
          ],
        },
      },
    }
    for e in std.mapWithIndex(function(i, v) {
      nickname: std.format('cass%03d', i + 1),
      host_name: deployment_name + '-' + self.nickname,
      token: cassandra_tokens[i],
      ip_address: v,
    }, cassandra_ips)
  },

  local daemon_instances = {
    [e.nickname]: {
      host_name: e.host_name,
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: e.ip_address,
      flavor: instance_flavor_daemon,
      image: instance_image_name,
      availability_zone: default_availability_zone,
      private_keys: [
        {
          name: '{CAPIDEPLOY_SFTP_USER}',
          private_key_path: sftp_config_private_key_path,
        },
      ],
      service: {
        env: {
          AMQP_URL: 'amqp://{CAPIDEPLOY_RABBITMQ_USER_NAME}:{CAPIDEPLOY_RABBITMQ_USER_PASS}@' + rabbitmq_ip + '/',
          CASSANDRA_HOSTS: cassandra_hosts,
          DAEMON_THREAD_POOL_SIZE: DEFAULT_DAEMON_THREAD_POOL_SIZE,
          DAEMON_DB_WRITERS: DEFAULT_DAEMON_DB_WRITERS,
          INTERNAL_BASTION_IP: internal_bastion_ip,
          NFS_DIRS: '/mnt/capi_cfg,/mnt/capi_in,/mnt/capi_out',
          PROMETHEUS_NODE_EXPORTER_VERSION: prometheus_node_exporter_version,
          SFTP_USER: '{CAPIDEPLOY_SFTP_USER}',
          SSH_USER: $.ssh_config.user,
        },
        cmd: {
          install: [
            'sh/common/replace_nameserver.sh',
            'sh/nfs/install_client.sh',
            "sh/daemon/install.sh",
            'sh/prometheus/install_node_exporter.sh',
          ],
          config: [
            'sh/nfs/config_client.sh',
            'sh/logrotate/config_capidaemon_logrotate.sh',
            'sh/rsyslog/config_capidaemon_log_sender.sh',
            'sh/prometheus/config_node_exporter.sh',
            'sh/daemon/config.sh',
          ],
          start: [
            'sh/daemon/start.sh',
          ],
          stop: [
            'sh/daemon/stop.sh',
          ],
        },
      },
      applicable_file_groups: [
        'up_daemon_binary',
        'up_daemon_env_config',
      ],
    }
    for e in std.mapWithIndex(function(i, v) {
      nickname: std.format('daemon%03d', i + 1),
      host_name: deployment_name + '-' + self.nickname,
      ip_address: v,
    }, daemon_ips)
  },

  instances: bastion_instance + rabbitmq_instance + prometheus_instance + cass_instances + daemon_instances,
}
