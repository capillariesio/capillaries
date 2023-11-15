{
  // Variables to play with

  // Choose your provider here. Openstack 002,003,004, AWS 005
  local dep_name = 'sampledeployment005',  // Can be any combination of alphanumeric characters. Make it unique.

  // x - test bare minimum, 2x - better, 4x - decent test, 16x - that's where it gets interesting
  local cassandra_node_flavor = 'aws.c6a.8',
  local architecture = 'amd64', // amd64 or arm64 
  // Cassandra cluster size - 4,8,16
  local cassandra_total_nodes = 4, 
  // If tasks are CPU-intensive (Python calc), make it equal to cassandra_total_nodes, otherwise cassandra_total_nodes/2
  local daemon_total_instances = cassandra_total_nodes, 
  local DEFAULT_DAEMON_THREAD_POOL_SIZE = '6', // daemon_cores*1.5
  local DEFAULT_DAEMON_DB_WRITERS = '16', // Depends on cassandra latency, reasonable values are 5-20

  // Basics
  local default_root_key_name = dep_name + '-root-key',  // This should match the name of the keypair you already created in Openstack/AWS

// Helper
  local provider_name = getFromMap({
    'sampledeployment002': 'openstack',
    'sampledeployment003': 'openstack',
    'sampledeployment004': 'openstack',
    'sampledeployment005': 'aws'
  }, dep_name),


  // Network
  // This is what external network is called for this cloud provider (used by Openstack)
  local external_gateway_network_name = getFromMap({
    'sampledeployment002': 'ext-net',
    'sampledeployment003': 'Ext-Net',
    'sampledeployment004': 'ext-floating1',
    'sampledeployment005': 'ext-network-not-needed-for-aws'
  }, dep_name),

  local vpc_cidr = '10.5.0.0/16', // AWS only
  local private_subnet_cidr = '10.5.0.0/24',
  local public_subnet_cidr = '10.5.1.0/24', // AWS only
  local private_subnet_allocation_pool = 'start=10.5.0.240,end=10.5.0.254',  // We use fixed ip addresses in the .0.2-.0.239 range, the rest is potentially available
  local bastion_subnet_type = if provider_name == 'aws' then 'public' else 'private',

  // Used by AWS only
  local subnet_availability_zone = getFromMap({
    'sampledeployment002': 'not-used-by-openstack',
    'sampledeployment003': 'not-used-by-openstack',
    'sampledeployment004': 'not-used-by-openstack',
    'sampledeployment005': 'us-east-1a'
  }, dep_name),

  // Internal IPs
  local internal_bastion_ip = if provider_name == 'aws' then '10.5.1.10' else '10.5.0.10', // In AWS, bastion is in the public subnet 10.5.1.0/24
  local prometheus_ip = '10.5.0.4',
  local rabbitmq_ip = '10.5.0.5',
  local daemon_ips = 
    if daemon_total_instances == 2 then ['10.5.0.101', '10.5.0.102']
    else if daemon_total_instances == 4 then ['10.5.0.101', '10.5.0.102', '10.5.0.103', '10.5.0.104']
    else if daemon_total_instances == 8 then ['10.5.0.101', '10.5.0.102', '10.5.0.103', '10.5.0.104', '10.5.0.105', '10.5.0.106', '10.5.0.107', '10.5.0.108']
    else if daemon_total_instances == 16 then ['10.5.0.101', '10.5.0.102', '10.5.0.103', '10.5.0.104', '10.5.0.105', '10.5.0.106', '10.5.0.107', '10.5.0.108', '10.5.0.109', '10.5.0.110', '10.5.0.111', '10.5.0.112', '10.5.0.113', '10.5.0.114', '10.5.0.115', '10.5.0.116']
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
  local cassandra_seeds = std.join(',', cassandra_ips),  // Used by cassandra nodes, all are seeds to avoid bootstrapping
  local cassandra_hosts = "'[\"" + std.join('","', cassandra_ips) + "\"]'",  // Used by daemons "'[\"10.5.0.11\",\"10.5.0.12\",\"10.5.0.13\",\"10.5.0.14\",\"10.5.0.15\",\"10.5.0.16\",\"10.5.0.17\",\"10.5.0.18\"]'",
  
  // Instances
  local instance_availability_zone = getFromMap({
    'sampledeployment002': 'us-central-1a',
    'sampledeployment003': 'nova',
    'sampledeployment004': 'dc3-a-09',
    'sampledeployment005': 'not-used-borrowed-from-subnet' // AWS borrows availability zone from the subnet
  }, dep_name),

  local instance_image_name = getFromMap({
    'sampledeployment002': 'ubuntu-23.04_LTS-lunar-server-cloudimg-amd64-20221217_raw',
    'sampledeployment003': 'Ubuntu 23.04',
    'sampledeployment004': 'Ubuntu 22.04 LTS Jammy Jellyfish',
    'sampledeployment005':
      if architecture == 'arm64' then 'ami-064b469793e32e5d2' // ubuntu/images/hvm-ssd/ubuntu-lunar-23.04-arm64-server-20230904
      else if architecture == 'amd64' then 'ami-0d8583a0d8d6dd14f' //ubuntu/images/hvm-ssd/ubuntu-lunar-23.04-amd64-server-20230714
      else 'unknown-architecture-unknown-image'
  }, dep_name),

  local instance_flavor_rabbitmq = getFromMap({
    'sampledeployment002': 't5sd.large',
    'sampledeployment003': 'b2-7',
    'sampledeployment004': 'a1-ram2-disk20-perf1',
    'sampledeployment005':
      if architecture == 'arm64' then 'c7g.medium'
      else if architecture == 'amd64' then 't2.micro'
      else 'unknown-architecture-unknown-rabbitmq-flavor'
  }, dep_name),

  local instance_flavor_prometheus = getFromMap({
    'sampledeployment002': 't5sd.large',
    'sampledeployment003': 'b2-7',
    'sampledeployment004': 'a1-ram2-disk20-perf1',
    'sampledeployment005':
      if architecture == 'arm64' then 'c7g.medium'
      else if architecture == 'amd64' then 't2.micro'
      else 'unknown-architecture-unknown-prometheus-flavor'
  }, dep_name),

  // Something modest, but capable of serving as NFS server, Webapi, UI
  local instance_flavor_bastion = getFromDoubleMap({
    'sampledeployment002': {
      'x': 'c5sd.large',
      '2x': 'c5sd.large',
      '4x': 'c5sd.xlarge',
    },
    'sampledeployment003': {
      'x': 'b2-7'
    },
    'sampledeployment004': {
      'x': 'a1-ram2-disk20-perf1',
      '2x': 'a1-ram2-disk20-perf1',
      '4x': 'a1-ram2-disk20-perf1',
      '8x': 'a1-ram2-disk20-perf1',
      '16x': 'a1-ram2-disk20-perf1'
    },
    'sampledeployment005': {
      '4x': 'c6a.large',
      '16x': 'c6a.large',
      'aws.c6a.8': 'c6a.large',
      'aws.c6a.32': 'c6a.large',
      'aws.c6a.64': 'c6a.large',
      'aws.c7g.16': 'c7g.large',
      'aws.c7g.32': 'c7g.large',
      'aws.c7gn.32': 'c7g.large',
      'aws.c7gn.64': 'c7g.large',
      'aws.c7g.64': 'c7g.large',
      'aws.c7g.64.all.metal': 'c7g.large',
      'aws.hpc7g.64': 'c7g.large',
      '18x': 'c6a.large',
      '36x': 'c6a.large',
      '64x': 'c6a.large',
      '96x': 'c6a.large'
    }
  }, dep_name, cassandra_node_flavor),

  // Fast/big everything: CPU, network, disk, RAM. Preferably local disk, preferably bare metal 
  local instance_flavor_cassandra = getFromDoubleMap({
    'sampledeployment002': {
      'x': 'c5d.xlarge', //'c6asx.xlarge'
      '2x': 'c5d.2xlarge', //'c6asx.2xlarge'
      '4x': 'c5d.4xlarge' //'m5d.4xlarge'//'c6asx.4xlarge'
    },
    'sampledeployment003': {
      'x': 'b2-7'
    },
    'sampledeployment004': {
      'x': 'a2-ram4-disk20-perf1', // They don't have perf2 version
      '2x': 'a4-ram8-disk20-perf2',
      '4x': 'a8-ram16-disk20-perf2',
      '8x': 'a16-ram32-disk20-perf1',
      '16x': 'a32-ram64-disk20-perf2' // They don't have perf1
    },
    'sampledeployment005': {
      '4x': 'c6a.2xlarge',
      '16x': 'c6a.8xlarge',
      'aws.c6a.8': 'c6a.2xlarge',
      'aws.c6a.32': 'c6a.8xlarge',
      'aws.c6a.64': 'c6a.16xlarge',
      'aws.c7g.16': 'c7g.4xlarge',
      'aws.c7g.32': 'c7g.8xlarge',
      'aws.c7gn.32': 'c7gn.8xlarge',
      'aws.c7gn.64': 'c7gn.16xlarge',
      'aws.c7g.64': 'c7g.metal',
      'aws.c7g.64.all.metal': 'c7g.metal',
      'aws.hpc7g.64': 'hpc7g.16xlarge',      
      '18x': 'c5n.9xlarge',
      '36x': 'c5n.metal',
      '64x': 'c6a.32xlarge',
      '96x': 'c6a.metal',
    },
  }, dep_name, cassandra_node_flavor),

  // Fast/big CPU, network, RAM. Disk optional.
  local instance_flavor_daemon = getFromDoubleMap({
    'sampledeployment002': {
      'x': 'c6sd.large',
      '2x': 'c6sd.xlarge',
      '4x': 'c6sd.2xlarge'
    },
    'sampledeployment003': {
      'x': 'b2-7'
    },
    'sampledeployment004': {
      'x' : 'a2-ram4-disk20-perf1',
      '2x': 'a4-ram8-disk20-perf1',
      '4x': 'a8-ram16-disk20-perf1', // For cluster16, need to stay within 200 vCpu quota, so no a8-ram16 for daemons 
      '8x': 'a8-ram16-disk20-perf1', // For cluster16, need to stay within 200 vCpu quota, so no a8-ram16 for daemons 
      '16x': 'a16-ram32-disk20-perf1'
    },
    'sampledeployment005': {
      '4x': 'c6a.xlarge',
      '16x': 'c6a.4xlarge',
      'aws.c6a.8': 'c6a.xlarge',
      'aws.c6a.32': 'c6a.4xlarge',
      'aws.c6a.64': 'c6a.8xlarge',
      'aws.c7g.16': 'c7g.2xlarge',
      'aws.c7g.32': 'c7g.4xlarge',
      'aws.c7gn.32': 'c7gn.4xlarge',
      'aws.c7gn.64': 'c7gn.8xlarge',
      'aws.c7g.64': 'c7g.8xlarge',
      'aws.c7g.64.all.metal': 'c7g.metal',
      'aws.hpc7g.64': 'hpc7g.8xlarge',
      '18x': 'c5n.4xlarge',
      '36x': 'c5n.9xlarge',
      '64x': 'c6a.16xlarge',
      '96x': 'c6a.24xlarge'
    }
  }, dep_name, cassandra_node_flavor),

  // Volumes
  local volume_availability_zone = getFromMap({
    'sampledeployment002': instance_availability_zone,
    'sampledeployment003': 'nova',
    'sampledeployment004': 'nova',
    'sampledeployment005': subnet_availability_zone
  }, dep_name),

  // Something modest to store in/out data and cfg
  local volume_type = getFromMap({
    'sampledeployment002': 'gp1',
    'sampledeployment003': 'classic',
    'sampledeployment004': 'CEPH_1_perf1',
    'sampledeployment005': 'gp2'
  }, dep_name),
  
  // Artifacts
  local buildLinuxAmd64Dir = '../../build/linux/amd64',
  local buildLinuxArm64Dir = '../../build/linux/arm64',
  local buildLinuxDir =
    if architecture == 'amd64' then buildLinuxAmd64Dir
    else if architecture == 'arm64' then buildLinuxArm64Dir
    else 'unknown-architecture-unknown-build-dir',
  local pkgExeDir = '../../pkg/exe',
  
  // Keys
  local sftp_config_public_key_path = '~/.ssh/' + dep_name + '_sftp.pub',
  local sftp_config_private_key_path = '~/.ssh/' + dep_name + '_sftp',
  local ssh_config_private_key_path = '~/.ssh/' + dep_name + '_rsa',
  
  // Prometheus versions
  local prometheus_node_exporter_version = '1.6.0',
  local prometheus_server_version = '2.45.0',
  local prometheus_cassandra_exporter_version = '0.9.12',

  // Used by Prometheus "\\'localhost:9100\\',\\'10.5.0.10:9100\\',\\'10.5.0.5:9100\\',\\'10.5.0.11:9100\\'...",
  local prometheus_targets = std.format("\\'localhost:9100\\',\\'%s:9100\\',\\'%s:9100\\',", [internal_bastion_ip, rabbitmq_ip]) +
                             "\\'" + std.join(":9100\\',\\'", cassandra_ips) + ":9100\\'," +
                             "\\'" + std.join(":9500\\',\\'", cassandra_ips) + ":9500\\'," + // Cassandra exporter
                             "\\'" + std.join(":9100\\',\\'", daemon_ips) + ":9100\\'",

  deploy_provider_name: provider_name,

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
    // Used by Capideploy Openstack calls
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
    // Used by AWS CLI
    'AWS_ACCESS_KEY_ID',
    'AWS_SECRET_ACCESS_KEY',
    'AWS_DEFAULT_REGION',
  ],
  ssh_config: {
    external_ip_address: '',
    port: 22,
    user: '{CAPIDEPLOY_SSH_USER}',
    private_key_path: ssh_config_private_key_path,
    private_key_password: '{CAPIDEPLOY_SSH_PRIVATE_KEY_PASS}',
  },
  timeouts: {
    openstack_cmd: 120,
    openstack_instance_creation: 240,
    attach_volume: 60,
  },

  // It's unlikely that you need to change anything below this line

  artifacts: {
    env: {
      DIR_BUILD_LINUX_AMD64: '../../' + buildLinuxAmd64Dir,
      DIR_BUILD_LINUX_ARM64: '../../' + buildLinuxArm64Dir,
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
    name: dep_name + '_network',
    cidr: vpc_cidr,
    private_subnet: {
      name: dep_name + '_private_subnet',
      cidr: private_subnet_cidr,
      availability_zone: subnet_availability_zone,
      allocation_pool: private_subnet_allocation_pool,
    },
    public_subnet: {
      name: dep_name + '_public_subnet',
      cidr: public_subnet_cidr,
      availability_zone: subnet_availability_zone,
      nat_gateway_name: dep_name + '_natgw',
    },
    router: { // aka AWS internet gateway
      name: dep_name + '_router',
      external_gateway_network_name: external_gateway_network_name,
    },
  },
  security_groups: {
    bastion: {
      name: dep_name + '_bastion_security_group',
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
          remote_ip: $.network.cidr,
          port: 111,
          direction: 'ingress',
        },
        {
          desc: 'NFS PortMapper UDP',
          protocol: 'udp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 111,
          direction: 'ingress',
        },
        {
          desc: 'NFS Server TCP',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 2049,
          direction: 'ingress',
        },
        {
          desc: 'NFS Server UDP',
          protocol: 'udp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
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
          remote_ip: $.network.cidr,
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
          remote_ip: $.network.cidr,
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
      name: dep_name + '_internal_security_group',
      rules: [
        {
          desc: 'SSH',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 22,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus UI internal',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 9090,
          direction: 'ingress',
        },
        {
          desc: 'Prometheus node exporter',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 9100,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra Prometheus node exporter',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 9500,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra JMX',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 7199,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra cluster comm',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 7000,
          direction: 'ingress',
        },
        {
          desc: 'Cassandra API',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 9042,
          direction: 'ingress',
        },
        {
          desc: 'RabbitMQ API',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
          port: 5672,
          direction: 'ingress',
        },
        {
          desc: 'RabbitMQ UI',
          protocol: 'tcp',
          ethertype: 'IPv4',
          remote_ip: $.network.cidr,
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
      src: buildLinuxDir + '/capiparquet.gz',
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
      src: buildLinuxDir + '/capidaemon.gz',
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
    up_portfolio_bigtest_in: {
      src: '/tmp/capi_in/portfolio_bigtest/all.tgz',
      dst: '/mnt/capi_in/portfolio_bigtest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {
        env: {
          OWNER_USER: $.ssh_config.user,
        },
        cmd: [
          'sh/capiscripts/unpack_portfolio_big_in.sh',
        ],
      },
    },
    up_portfolio_bigtest_out: {
      src: '/tmp/capi_out/portfolio_bigtest/all.tgz',
      dst: '/mnt/capi_out/portfolio_bigtest',
      dir_permissions: 777,
      file_permissions: 666,
      owner: $.ssh_config.user,
      after: {
        env: {
          OWNER_USER: $.ssh_config.user,
        },
        cmd: [
          'sh/capiscripts/unpack_portfolio_big_out.sh',
        ],
      },
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
      src: buildLinuxDir + '/capitoolbelt.gz',
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
      src: buildLinuxDir + '/capiwebapi.gz',
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
      host_name: dep_name + '-bastion',
      security_group: 'bastion',
      root_key_name: default_root_key_name,
      ip_address: internal_bastion_ip,
      uses_ssh_config_external_ip_address: true,
      flavor: instance_flavor_bastion,
      image: instance_image_name,
      availability_zone: instance_availability_zone,
      subnet_type: bastion_subnet_type,
      volumes: {
        cfg: {
          name: dep_name + '_cfg',
          availability_zone: volume_availability_zone,
          mount_point: '/mnt/capi_cfg',
          size: 1,
          type: volume_type,
          permissions: 777,
          owner: $.ssh_config.user, // If SFTP used: "{CAPIDEPLOY_SFTP_USER}"
        },
        'in': {
          name: dep_name + '_in',
          availability_zone: volume_availability_zone,
          mount_point: '/mnt/capi_in',
          size: 1,
          type: volume_type,
          permissions: 777,
          owner: $.ssh_config.user,
        },
        out: {
          name: dep_name + '_out',
          availability_zone: volume_availability_zone,
          mount_point: '/mnt/capi_out',
          size: 1,
          type: volume_type,
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
          NETWORK_CIDR: $.network.cidr,
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
        'up_portfolio_bigtest_in',
        'up_portfolio_bigtest_out',
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
      host_name: dep_name + '-rabbitmq',
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: rabbitmq_ip,
      flavor: instance_flavor_rabbitmq,
      image: instance_image_name,
      availability_zone: instance_availability_zone,
      subnet_type: 'private',
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
      host_name: dep_name + '-prometheus',
      security_group: 'internal',
      root_key_name: default_root_key_name,
      ip_address: prometheus_ip,
      flavor: instance_flavor_prometheus,
      image: instance_image_name,
      availability_zone: instance_availability_zone,
      subnet_type: 'private',
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
      availability_zone: instance_availability_zone,
      subnet_type: 'private',
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
      host_name: dep_name + '-' + self.nickname,
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
      availability_zone: instance_availability_zone,
      subnet_type: 'private',
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
            'sh/rsyslog/restart.sh', // It's stupid, but on AWS machines it's required, otherwise capidaemon.log is notpicked up whenit appears.
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
      host_name: dep_name + '-' + self.nickname,
      ip_address: v,
    }, daemon_ips)
  },

  instances: bastion_instance + rabbitmq_instance + prometheus_instance + cass_instances + daemon_instances,

  local getFromMap = function(m, k)
    if std.length(m[k]) > 0 then m[k] else "no-key-" + k,

  local getFromDoubleMap = function(m, k1, k2)
    if std.length(m[k1]) > 0 then 
      if std.length(m[k1][k2]) > 0 then m[k1][k2] else "no-key-" + k2
    else  "no-key-" + k1,
}

