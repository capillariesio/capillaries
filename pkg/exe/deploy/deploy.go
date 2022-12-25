package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/deploy"
)

const (
	CmdCreateSecurityGroup string = "create_security_group"
	CmdDeleteSecurityGroup string = "delete_security_group"
	CmdCreateNetworking    string = "create_networking"
	CmdDeleteNetworking    string = "delete_networking"
	CmdCreateVolumes       string = "create_volumes"
	CmdDeleteVolumes       string = "delete_volumes"
	CmdCreateInstances     string = "create_instances"
	CmdDeleteInstances     string = "delete_instances"
	CmdAttachVolumes       string = "attach_volumes"
	CmdUploadFiles         string = "upload_files"
	CmdDownloadFiles       string = "download_files"
	CmdSetupServices       string = "setup_services"
	CmdStartServices       string = "start_services"
	CmdStopServices        string = "stop_services"
)

type SingleThreadCmdHandler func(*deploy.ProjectPair, chan string) error

func DumpLogChan(logChan chan string, isVerbose bool) {
	for len(logChan) > 0 {
		msg := <-logChan
		if isVerbose {
			fmt.Println(msg)
		}
	}
}

func FilterByNickname[GenericDef deploy.FileGroupUpDef | deploy.FileGroupDownDef | deploy.InstanceDef](sourceMap map[string]*GenericDef) (map[string]*GenericDef, error) {
	var defMap map[string]*GenericDef
	if os.Args[2] == "*" {
		defMap = sourceMap
	} else {
		defMap = map[string]*GenericDef{}
		for _, defNickname := range strings.Split(os.Args[2], ",") {
			fgDef, ok := sourceMap[defNickname]
			if !ok {
				return nil, fmt.Errorf("definition for %s not found", defNickname)
			}
			defMap[defNickname] = fgDef
		}
	}
	return defMap, nil
}

func main() {
	prjPair, fullPrjPath, err := deploy.LoadProject("deploy.json", "deploy_params.json")
	if err != nil {
		log.Fatalf(err.Error())
	}

	if len(os.Args) <= 1 {
		log.Fatalf("COMMAND EXPECTED")
	}

	cmdStartTs := time.Now()

	commonArgs := flag.NewFlagSet("common args", flag.ExitOnError)
	argVerbosity := commonArgs.Bool("verbose", false, "Verbosity")

	const MaxWorkerThreads int = 10
	var logChan = make(chan string, MaxWorkerThreads*5)
	var sem = make(chan int, MaxWorkerThreads)
	var errChan chan error
	errorsExpected := 1

	singleThreadCommands := map[string]SingleThreadCmdHandler{
		CmdCreateSecurityGroup: deploy.CreateSecurityGroup,
		CmdDeleteSecurityGroup: deploy.DeleteSecurityGroup,
		CmdCreateNetworking:    deploy.CreateNetworking,
		CmdDeleteNetworking:    deploy.DeleteNetworking,
	}

	if cmdHandler, ok := singleThreadCommands[os.Args[1]]; ok {
		commonArgs.Parse(os.Args[2:])
		errChan = make(chan error, errorsExpected)
		sem <- 1
		go func() {
			errChan <- cmdHandler(prjPair, logChan)
			<-sem
		}()
	} else if os.Args[1] == CmdCreateInstances || os.Args[1] == CmdDeleteInstances {
		commonArgs.Parse(os.Args[3:])
		if len(os.Args[2]) == 0 {
			log.Fatalf("expected comma-separated list of instances or *")
		}
		instances, err := FilterByNickname(prjPair.Live.Instances)
		if err != nil {
			log.Fatalf(err.Error())
		}
		errorsExpected = len(instances)
		errChan = make(chan error, errorsExpected)
		switch os.Args[1] {
		case CmdCreateInstances:
			// Make sure image/flavor is supported
			usedFlavors := map[string]string{}
			usedImages := map[string]string{}
			for _, instDef := range instances {
				usedFlavors[instDef.FlavorName] = ""
				usedImages[instDef.ImageName] = ""
			}
			if err := deploy.GetFlavorIds(prjPair, logChan, usedFlavors); err != nil {
				log.Fatalf(err.Error())
			}
			DumpLogChan(logChan, *argVerbosity)

			if err := deploy.GetImageIds(prjPair, logChan, usedImages); err != nil {
				log.Fatalf(err.Error())
			}
			DumpLogChan(logChan, *argVerbosity)

			for iNickname, _ := range instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string) {
					errChan <- deploy.CreateInstance(prjPair, logChan, iNickname, usedFlavors[prjPair.Live.Instances[iNickname].FlavorName], usedImages[prjPair.Live.Instances[iNickname].ImageName])
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		case CmdDeleteInstances:
			for iNickname, _ := range instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string) {
					errChan <- deploy.DeleteInstance(prjPair, logChan, iNickname)
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		default:
			log.Fatalf("unknown create/delete instance command:" + os.Args[1])
		}
	} else if os.Args[1] == CmdSetupServices || os.Args[1] == CmdStartServices || os.Args[1] == CmdStopServices {
		commonArgs.Parse(os.Args[3:])
		if len(os.Args[2]) == 0 {
			log.Fatalf("expected comma-separated list of instances or *")
		}

		instances, err := FilterByNickname(prjPair.Live.Instances)
		if err != nil {
			log.Fatalf(err.Error())
		}

		errorsExpected = len(instances)
		errChan = make(chan error, len(instances))
		for _, iDef := range instances {
			sem <- 1
			go func(prj *deploy.Project, logChan chan string, errChan chan error, iDef *deploy.InstanceDef) {
				var cmdToRun []string
				switch os.Args[1] {
				case CmdSetupServices:
					cmdToRun = iDef.Service.Cmd.Setup
				case CmdStartServices:
					cmdToRun = iDef.Service.Cmd.Start
				case CmdStopServices:
					cmdToRun = iDef.Service.Cmd.Stop
				default:
					log.Fatalf("unknown setup/start/stop service command:" + os.Args[1])
				}
				errChan <- deploy.ExecScriptsOnInstance(prj, logChan, iDef.BestIpAddress(), iDef.Service.Env, cmdToRun)
				<-sem
			}(&prjPair.Live, logChan, errChan, iDef)
		}
	} else {
		switch os.Args[1] {

		case CmdCreateVolumes:
			commonArgs.Parse(os.Args[2:])
			errorsExpected = len(prjPair.Live.Volumes)
			errChan = make(chan error, errorsExpected)
			for volNickname, _ := range prjPair.Live.Volumes {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, volNickname string) {
					errChan <- deploy.CreateVolume(prjPair, logChan, volNickname)
					<-sem
				}(prjPair, logChan, errChan, volNickname)
			}

		case CmdDeleteVolumes:
			commonArgs.Parse(os.Args[2:])
			errorsExpected = len(prjPair.Live.Volumes)
			errChan = make(chan error, errorsExpected)
			for volNickname, _ := range prjPair.Live.Volumes {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, volNickname string) {
					errChan <- deploy.DeleteVolume(prjPair, logChan, volNickname)
					<-sem
				}(prjPair, logChan, errChan, volNickname)
			}

		case CmdAttachVolumes:
			commonArgs.Parse(os.Args[3:])
			if len(os.Args[2]) == 0 {
				log.Fatalf("expected comma-separated list of instances or *")
			}

			instances, err := FilterByNickname(prjPair.Live.Instances)
			if err != nil {
				log.Fatalf(err.Error())
			}

			attachmentCount := 0
			for iNickname, iDef := range instances {
				for volNickname, _ := range iDef.AttachedVolumes {
					if _, ok := prjPair.Live.Volumes[volNickname]; !ok {
						log.Fatalf("cannot find volume %s referenced in instance %s", volNickname, iNickname)
					}
					attachmentCount++
				}
			}
			errorsExpected = attachmentCount
			errChan = make(chan error, attachmentCount)
			for iNickname, iDef := range instances {
				for volNickname, _ := range iDef.AttachedVolumes {
					sem <- 1
					go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string, volNickname string) {
						errChan <- deploy.AttachVolume(prjPair, logChan, iNickname, volNickname)
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				}
			}

		case CmdUploadFiles:
			commonArgs.Parse(os.Args[3:])
			if len(os.Args[2]) == 0 {
				log.Fatalf("expected comma-separated list of file groups to upload or *")
			}

			fileGroups, err := FilterByNickname(prjPair.Live.FileGroupsUp)
			if err != nil {
				log.Fatalf(err.Error())
			}

			// Walk through src locally and create file upload specs
			fileSpecs, err := deploy.FileGroupUpDefsToSpecs(&prjPair.Live, fileGroups)
			if err != nil {
				log.Fatalf(err.Error())
			}

			errorsExpected = len(fileSpecs)
			errChan = make(chan error, len(fileSpecs))
			for _, fuSpec := range fileSpecs {
				sem <- 1
				go func(prj *deploy.Project, logChan chan string, errChan chan error, fuSpec *deploy.FileUploadSpec) {
					errChan <- deploy.UploadFileSftp(prj, logChan, fuSpec.IpAddress, fuSpec.Src, fuSpec.Dst, fuSpec.Permissions)
					<-sem
				}(&prjPair.Live, logChan, errChan, fuSpec)
			}

		case CmdDownloadFiles:
			commonArgs.Parse(os.Args[3:])
			if len(os.Args[2]) == 0 {
				log.Fatalf("expected comma-separated list of file groups to download or *")
			}

			fileGroups, err := FilterByNickname(prjPair.Live.FileGroupsDown)
			if err != nil {
				log.Fatalf(err.Error())
			}

			// Walk through src remotely and create file upload specs
			fileSpecs, err := deploy.FileGroupDownDefsToSpecs(&prjPair.Live, fileGroups)
			if err != nil {
				log.Fatalf(err.Error())
			}

			errorsExpected = len(fileSpecs)
			errChan = make(chan error, len(fileSpecs))
			for _, fdSpec := range fileSpecs {
				sem <- 1
				go func(prj *deploy.Project, logChan chan string, errChan chan error, fdSpec *deploy.FileDownloadSpec) {
					errChan <- deploy.DownloadFileSftp(prj, logChan, fdSpec.IpAddress, fdSpec.Src, fdSpec.Dst)
					<-sem
				}(&prjPair.Live, logChan, errChan, fdSpec)
			}

		default:
			log.Fatalf("unknown command:" + os.Args[1])
		}
	}

	finalCmdErr := 0
	for errorsExpected > 0 {
		select {
		case cmdErr := <-errChan:
			if cmdErr != nil {
				finalCmdErr = 1
				fmt.Fprintf(os.Stderr, "%s\n", cmdErr.Error())
			}
			errorsExpected--
		case msg := <-logChan:
			if *argVerbosity {
				fmt.Println(msg)
			}
		}
	}

	DumpLogChan(logChan, *argVerbosity)

	if finalCmdErr > 0 {
		os.Exit(finalCmdErr)
	}

	// Save updated project template, it may have some new ids and timestamps
	if err = prjPair.Template.SaveProject(fullPrjPath); err != nil {
		log.Fatalf(err.Error())
	}
	if *argVerbosity {
		fmt.Printf("Elapsed: %.3fs\n", time.Since(cmdStartTs).Seconds())
	}
}

// reserve_floating_ips
// create_security_group
// create_networking
// create_volumes (gets list of all volumes in the sec group and reports if there are extra running, checking by name)
// create_instances: (gets list of all instances in the sec group and reports if there are extra running, checking by name)
// - also assigns floating ip if needed

// attach_volumes

// setup_instances (non-idempotent) (requires list/mask of instances, may be used when adding more instances)

// start_services

// upload_files (requires list of groups)
// ... run toolbelt commands or UI/Webapi
// download_files (requires list of groups)

// stop_services

// delete_instances (list/mask of instances)
// delete_volumes (list volumes)
// delete_networking
// delete_security_group
// release_floating_ips
