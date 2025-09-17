package downdetect

var downdetectService = &DowndetectService{}
var downdetectController = &DowndetectController{
	downdetectService,
}

func GetDowndetectController() *DowndetectController {
	return downdetectController
}
