package vlib

type DockerVersion struct {
	Client struct {
		Version    string
		APIVersion string
		GoVersion  string
		GitCommit  string
		Built      string
		OSArch     string
		Context    string
	}

	Server struct {
		Engine struct {
			Version      string
			APIVersion   string
			GoVersion    string
			GitCommit    string
			Built        string
			OSArch       string
			Experimental bool
		}

		Containerd struct {
			Version   string
			GitCommit string
		}

		Runc struct {
			Version   string
			GitCommit string
		}

		TiniStatic struct {
			Version   string
			GitCommit string
		}
	}
}

type CaddyVersion struct {
	Version     string
	VersionFull string
}

type AsteriskVersion struct {
	Version     string
	VersionFull string
}

type FrankenPHPVersion struct {
	FrankenPHP struct {
		Version     string
		VersionFull string
	}
	PHP struct {
		Version     string
		VersionFull string
	}
	Caddy struct {
		Version     string
		VersionFull string
	}
	VersionFull string
}

type HAProxyVersion struct {
	Version     string
	Status      string
	KnownBugs   string
	RunningOn   string
	VersionFull string
}

type JenkinsVersion struct {
	Version     string
	VersionFull string
}

type MongoDBVersion struct {
	Environment struct {
		Distmod    string `json:"distmod"`
		Distarch   string `json:"distarch"`
		TargetArch string `json:"target_arch"`
	} `json:"environment"`
	Version        string   `json:"version"`
	VersionFull    string   `json:"-"`
	GitVersion     string   `json:"gitVersion"`
	OpenSSLVersion string   `json:"openSSLVersion"`
	Modules        []string `json:"modules"`
	Allocator      string   `json:"allocator"`
}

type MySQLVersion struct {
	Version     string
	VersionFull string
}

type MariaDBVersion struct {
	Version     string
	VersionFull string
}

type NginxVersion struct {
	Version     string
	VersionFull string
}

type OPNsenseVersion struct {
	Version     string
	VersionFull string
}

type PostalVersion struct {
	Version     string
	VersionFull string
}

type PostgreSQLVersion struct {
	Version     string
	VersionFull string
}

type RedisVersion struct {
	Version     string
	VersionFull string
}

type ValkeyVersion struct {
	Version     string
	VersionFull string
}

type VaultVersion struct {
	Version     string
	VersionFull string
}

type RabbitMQVersion struct {
	Memory Memory `json:"memory"`

	PID      int      `json:"pid"`
	RunQueue int      `json:"run_queue"`
	OS       string   `json:"os"`
	NetTick  int      `json:"net_ticktime"`
	Alarms   []string `json:"alarms"`

	Processes Processes  `json:"processes"`
	Listeners []Listener `json:"listeners"`

	ProductName    string `json:"product_name"`
	ProductVersion string `json:"product_version"`

	ErlangVersion      string `json:"erlang_version"`
	RabbitMQVersion    string `json:"rabbitmq_version"`
	Uptime             int    `json:"uptime"`
	IsUnderMaintenance bool   `json:"is_under_maintenance"`

	CryptoLibVersion string `json:"crypto_lib_version"`

	EnabledPluginFile string   `json:"enabled_plugin_file"`
	ActivePlugins     []string `json:"active_plugins"`

	DataDirectory     string   `json:"data_directory"`
	RaftDataDirectory string   `json:"raft_data_directory"`
	ConfigFiles       []string `json:"config_files"`
	LogFiles          []string `json:"log_files"`

	VMMemoryCalculationStrategy  string                       `json:"vm_memory_calculation_strategy"`
	VMMemoryHighWatermarkSetting VMMemoryHighWatermarkSetting `json:"vm_memory_high_watermark_setting"`
	VMMemoryHighWatermarkLimit   int64                        `json:"vm_memory_high_watermark_limit"`

	FileDescriptors FileDescriptors `json:"file_descriptors"`

	DiskFreeLimit int64 `json:"disk_free_limit"`
	DiskFree      int64 `json:"disk_free"`

	Totals Totals `json:"totals"`

	ReleaseSeriesSupportStatus bool `json:"release_series_support_status"`

	Version     string `json:"version"`
	VersionFull string `json:"version_full"`
}

type Memory struct {
	Atom   int64 `json:"atom"`
	Binary int64 `json:"binary"`
	Code   int64 `json:"code"`

	Total MemoryTotal `json:"total"`

	Strategy string `json:"strategy"`

	Plugins int64 `json:"plugins"`
	Mnesia  int64 `json:"mnesia"`

	ConnectionReaders  int64 `json:"connection_readers"`
	ConnectionWriters  int64 `json:"connection_writers"`
	ConnectionChannels int64 `json:"connection_channels"`
	ConnectionOther    int64 `json:"connection_other"`

	QueueProcs      int64 `json:"queue_procs"`
	QueueSlaveProcs int64 `json:"queue_slave_procs"`

	QuorumQueueProcs    int64 `json:"quorum_queue_procs"`
	QuorumQueueDlxProcs int64 `json:"quorum_queue_dlx_procs"`

	StreamQueueProcs              int64 `json:"stream_queue_procs"`
	StreamQueueReplicaReaderProcs int64 `json:"stream_queue_replica_reader_procs"`
	StreamQueueCoordinatorProcs   int64 `json:"stream_queue_coordinator_procs"`

	MetadataStore    int64 `json:"metadata_store"`
	MetadataStoreETS int64 `json:"metadata_store_ets"`
	OtherProc        int64 `json:"other_proc"`
	OtherETS         int64 `json:"other_ets"`
	OtherSystem      int64 `json:"other_system"`
	Metrics          int64 `json:"metrics"`
	MsgIndex         int64 `json:"msg_index"`
	MgmtDB           int64 `json:"mgmt_db"`
	QuorumETS        int64 `json:"quorum_ets"`

	AllocatedUnused     int64 `json:"allocated_unused"`
	ReservedUnallocated int64 `json:"reserved_unallocated"`
}

type MemoryTotal struct {
	Erlang    int64 `json:"erlang"`
	RSS       int64 `json:"rss"`
	Allocated int64 `json:"allocated"`
}

type Processes struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
}

type Listener struct {
	Node      string `json:"node"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Interface string `json:"interface"`
	Purpose   string `json:"purpose"`
}

type VMMemoryHighWatermarkSetting struct {
	Relative float64 `json:"relative"`
}

type FileDescriptors struct {
	TotalUsed    int `json:"total_used"`
	TotalLimit   int `json:"total_limit"`
	SocketsUsed  int `json:"sockets_used"`
	SocketsLimit int `json:"sockets_limit"`
}

type Totals struct {
	VirtualHostCount int `json:"virtual_host_count"`
	ConnectionCount  int `json:"connection_count"`
	QueueCount       int `json:"queue_count"`
}

type PrometheusVersion struct {
	Version     string
	BuildUser   string
	BuildDate   string
	GoVersion   string
	Platform    string
	Tags        []string
	VersionFull string
}

type ZabbixVersion struct {
	Version     string
	VersionFull string
}

type PVEVersion struct {
	Version     string
	VersionFull string
	Type        string
}

type ZimbraVersion struct {
	Version     string
	VersionFull string
}
