package main

const (
	pgContainerName    = "orca-pg-instance"
	redisContainerName = "orca-redis-instance"
	orcaContainerName  = "orca-instance"
	networkName        = "orca-network"
	orcaInternalPort   = 3335
	pgInternalPort     = 5432
	redisInternalPort  = 6379

	// versions
	orcaImageVersion = "0.11.0-rc.a2238d2"
)

var orcaContainers = []string{
	pgContainerName,
	redisContainerName,
	orcaContainerName,
}

// follows pattern of <container-name>-data
var orcaVolumes = []string{
	"orca-pg-instance-data",
	"orca-redis-instance-data",
}
