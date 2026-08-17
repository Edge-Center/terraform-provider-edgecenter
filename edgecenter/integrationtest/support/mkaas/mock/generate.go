package mkaasmock

//go:generate go run github.com/vektra/mockery/v2 --name=MKaaSService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=mkaasmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=TasksService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=mkaasmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ProjectsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=mkaasmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=RegionsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=mkaasmock --testonly=false --with-expecter=false --log-level=error
