package disk

var (
	diskService    *DiskService
	diskController *DiskController
)

func init() {
	diskService = &DiskService{}

	diskController = &DiskController{
		diskService,
	}
}

func GetDiskService() *DiskService {
	return diskService
}

func GetDiskController() *DiskController {
	return diskController
}
