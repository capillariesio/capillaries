package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

func main() {
	prjPair, fullPrjPath, err := deploy.LoadProject("deploy.json", "deploy_params.json")
	if err != nil {
		log.Fatalf(err.Error())
	}

	if len(os.Args) <= 1 {
		log.Fatalf("COMMAND EXPECTED")
	}

	commonArgs := flag.NewFlagSet("common args", flag.ExitOnError)
	argVerbosity := commonArgs.Bool("verbose", false, "Verbosity")
	commonArgs.Parse(os.Args[2:])

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
		errChan = make(chan error, errorsExpected)
		sem <- 1
		go func() {
			errChan <- cmdHandler(prjPair, logChan)
			<-sem
		}()
	} else {
		switch os.Args[1] {
		case CmdCreateVolumes:
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
			errorsExpected = len(prjPair.Live.Volumes)
			errChan = make(chan error, errorsExpected)
			for volNickname, _ := range prjPair.Live.Volumes {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, volNickname string) {
					errChan <- deploy.DeleteVolume(prjPair, logChan, volNickname)
					<-sem
				}(prjPair, logChan, errChan, volNickname)
			}
		case CmdCreateInstances:
			usedFlavors := map[string]string{}
			usedImages := map[string]string{}
			for _, instDef := range prjPair.Live.Instances {
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

			errorsExpected = len(prjPair.Live.Instances)
			errChan = make(chan error, errorsExpected)
			for iNickname, _ := range prjPair.Live.Instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string) {
					errChan <- deploy.CreateInstance(prjPair, logChan, iNickname, usedFlavors[prjPair.Live.Instances[iNickname].FlavorName], usedImages[prjPair.Live.Instances[iNickname].ImageName])
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		case CmdDeleteInstances:
			errorsExpected = len(prjPair.Live.Instances)
			errChan = make(chan error, errorsExpected)
			for iNickname, _ := range prjPair.Live.Instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string) {
					errChan <- deploy.DeleteInstance(prjPair, logChan, iNickname)
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		case CmdAttachVolumes:
			attachmentCount := 0
			for iNickname, iDef := range prjPair.Live.Instances {
				for volNickname, _ := range iDef.AttachedVolumes {
					if _, ok := prjPair.Live.Volumes[volNickname]; !ok {
						log.Fatalf("cannot find volume %s referenced in instance %s", volNickname, iNickname)
					}
					attachmentCount++
				}
			}
			errorsExpected = attachmentCount
			errChan = make(chan error, attachmentCount)
			for iNickname, iDef := range prjPair.Live.Instances {
				for volNickname, _ := range iDef.AttachedVolumes {
					sem <- 1
					go func(prjPair *deploy.ProjectPair, logChan chan string, errChan chan error, iNickname string, volNickname string) {
						errChan <- deploy.AttachVolume(prjPair, logChan, iNickname, volNickname)
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				}
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
