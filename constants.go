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
	orcaImageVersion = "0.12.0-rc.71dee4e"
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
