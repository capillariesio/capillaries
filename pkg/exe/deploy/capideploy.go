package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/deploy"
)

const (
	CmdCreateSecurityGroups string = "create_security_groups"
	CmdDeleteSecurityGroups string = "delete_security_groups"
	CmdCreateNetworking     string = "create_networking"
	CmdDeleteNetworking     string = "delete_networking"
	CmdCreateVolumes        string = "create_volumes"
	CmdDeleteVolumes        string = "delete_volumes"
	CmdCreateInstances      string = "create_instances"
	CmdDeleteInstances      string = "delete_instances"
	CmdAttachVolumes        string = "attach_volumes"
	CmdUploadFiles          string = "upload_files"
	CmdDownloadFiles        string = "download_files"
	CmdSetupServices        string = "setup_services"
	CmdStartServices        string = "start_services"
	CmdStopServices         string = "stop_services"
	CmdCreateInstanceUsers  string = "create_instance_users"
	CmdCopyPrivateKeys      string = "copy_private_keys"
	CmdPingInstances        string = "ping_instances"
)

type SingleThreadCmdHandler func(*deploy.ProjectPair, bool) (deploy.LogMsg, error)

func DumpLogChan(logChan chan deploy.LogMsg) {
	for len(logChan) > 0 {
		msg := <-logChan
		fmt.Println(string(msg))
	}
}

func getNicknamesArg(commonArgs *flag.FlagSet, entityName string) (string, error) {
	if len(os.Args) < 3 {
		return "", fmt.Errorf("not enough args, expected comma-separated list of %s or '*'", entityName)
	}
	commonArgs.Parse(os.Args[3:])
	if len(os.Args[2]) == 0 {
		return "", fmt.Errorf("bad arg, expected comma-separated list of %s or '*'", entityName)
	}
	return os.Args[2], nil
}

func filterByNickname[GenericDef deploy.FileGroupUpDef | deploy.FileGroupDownDef | deploy.InstanceDef](nicknames string, sourceMap map[string]*GenericDef, entityName string) (map[string]*GenericDef, error) {
	var defMap map[string]*GenericDef
	if nicknames == "all" {
		defMap = sourceMap
	} else {
		defMap = map[string]*GenericDef{}
		for _, defNickname := range strings.Split(nicknames, ",") {
			fgDef, ok := sourceMap[defNickname]
			if !ok {
				return nil, fmt.Errorf("definition for %s '%s' not found, available definitions: %s", entityName, defNickname, reflect.ValueOf(sourceMap).MapKeys())
			}
			defMap[defNickname] = fgDef
		}
	}
	return defMap, nil
}

func waitForWorkers(errorsExpected int, errChan chan error, logChan chan deploy.LogMsg) int {
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
			fmt.Println(msg)
		}
	}

	DumpLogChan(logChan)

	return finalCmdErr
}

func usage(flagset *flag.FlagSet) {
	fmt.Printf(`
Capillaries deploy
Usage: capideploy <command> [command parameters] [optional parameters]

Commands:
  %s
  %s
  %s
  %s
  %s
  %s
  %s <comma-separated list of instances to create, or 'all'>
  %s <comma-separated list of instances to delete, or 'all'>
  %s <comma-separated list of instances to ping, or 'all'>
  %s <comma-separated list of instances to create users on, or 'all'>
  %s <comma-separated list of instances to copy private keys to, or 'all'>
  %s <comma-separated list of instances to attach volumes to, or 'all'>
  %s <comma-separated list of upload file groups, or 'all'>
  %s <comma-separated list of download file groups, or 'all'>  
  %s <comma-separated list of instances to setup services on, or 'all'>
  %s <comma-separated list of instances to start services on, or 'all'>
  %s <comma-separated list of instances to stop services on, or 'all'>
`,
		CmdCreateSecurityGroups,
		CmdDeleteSecurityGroups,
		CmdCreateNetworking,
		CmdDeleteNetworking,
		CmdCreateVolumes,
		CmdDeleteVolumes,

		CmdCreateInstances,
		CmdDeleteInstances,
		CmdPingInstances,

		CmdCreateInstanceUsers,
		CmdCopyPrivateKeys,

		CmdAttachVolumes,

		CmdUploadFiles,
		CmdDownloadFiles,

		CmdSetupServices,
		CmdStartServices,
		CmdStopServices,
	)
	fmt.Printf("\nOptional parameters:\n")
	flagset.PrintDefaults()
	os.Exit(0)
}

func main() {
	commonArgs := flag.NewFlagSet("common args", flag.ExitOnError)
	argVerbosity := commonArgs.Bool("verbose", false, "Debug output")
	argPrjFile := commonArgs.String("prj", "capideploy_project.json", "Project file, looked in exe path, current dir")
	argPrjParamsFile := commonArgs.String("prj_params", "capideploy_project_params.json", "Project params file, looked in exe path, current dir, home dir")

	if len(os.Args) <= 1 {
		usage(commonArgs)
		os.Exit(1)
	}

	cmdStartTs := time.Now()

	prjPair, fullPrjPath, err := deploy.LoadProject(*argPrjFile, *argPrjParamsFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	const MaxWorkerThreads int = 10
	var logChan = make(chan deploy.LogMsg, MaxWorkerThreads*5)
	var sem = make(chan int, MaxWorkerThreads)
	var errChan chan error
	errorsExpected := 1

	singleThreadCommands := map[string]SingleThreadCmdHandler{
		CmdCreateSecurityGroups: deploy.CreateSecurityGroups,
		CmdDeleteSecurityGroups: deploy.DeleteSecurityGroups,
		CmdCreateNetworking:     deploy.CreateNetworking,
		CmdDeleteNetworking:     deploy.DeleteNetworking,
	}

	if cmdHandler, ok := singleThreadCommands[os.Args[1]]; ok {
		commonArgs.Parse(os.Args[2:])
		errChan = make(chan error, errorsExpected)
		sem <- 1
		go func() {
			logMsg, err := cmdHandler(prjPair, *argVerbosity)
			logChan <- logMsg
			errChan <- err
			<-sem
		}()
	} else if os.Args[1] == CmdCreateInstances || os.Args[1] == CmdDeleteInstances {
		nicknames, err := getNicknamesArg(commonArgs, "instances")
		if err != nil {
			log.Fatalf(err.Error())
		}
		instances, err := filterByNickname(nicknames, prjPair.Live.Instances, "instance")
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
			logMsg, err := deploy.GetFlavorIds(prjPair, usedFlavors, *argVerbosity)
			logChan <- logMsg
			DumpLogChan(logChan)
			if err != nil {
				log.Fatalf(err.Error())
			}

			logMsg, err = deploy.GetImageIds(prjPair, usedImages, *argVerbosity)
			logChan <- logMsg
			DumpLogChan(logChan)
			if err != nil {
				log.Fatalf(err.Error())
			}

			for iNickname, _ := range instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string) {
					logMsg, err := deploy.CreateInstance(prjPair, iNickname, usedFlavors[prjPair.Live.Instances[iNickname].FlavorName], usedImages[prjPair.Live.Instances[iNickname].ImageName], *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		case CmdDeleteInstances:
			for iNickname, _ := range instances {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string) {
					logMsg, err := deploy.DeleteInstance(prjPair, iNickname, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		default:
			log.Fatalf("unknown create/delete instance command:" + os.Args[1])
		}
	} else if os.Args[1] == CmdPingInstances || os.Args[1] == CmdCreateInstanceUsers || os.Args[1] == CmdCopyPrivateKeys || os.Args[1] == CmdSetupServices || os.Args[1] == CmdStartServices || os.Args[1] == CmdStopServices {
		nicknames, err := getNicknamesArg(commonArgs, "instances")
		if err != nil {
			log.Fatalf(err.Error())
		}

		instances, err := filterByNickname(nicknames, prjPair.Live.Instances, "instance")
		if err != nil {
			log.Fatalf(err.Error())
		}

		errorsExpected = len(instances)
		errChan = make(chan error, len(instances))
		for _, iDef := range instances {
			sem <- 1
			go func(prj *deploy.Project, logChan chan deploy.LogMsg, errChan chan error, iDef *deploy.InstanceDef) {
				var logMsg deploy.LogMsg
				var finalErr error
				switch os.Args[1] {
				case CmdPingInstances:
					// Just run WhoAmI
					logMsg, finalErr = deploy.ExecCommandsOnInstance(prjPair.Live.SshConfig, iDef.BestIpAddress(), []string{"id"}, *argVerbosity)
				case CmdCreateInstanceUsers:
					cmds, err := deploy.NewCreateInstanceUsersCommands(iDef)
					if err != nil {
						log.Fatalf("cannot build commands to create instance users: %s", err.Error())
					}
					logMsg, finalErr = deploy.ExecCommandsOnInstance(prjPair.Live.SshConfig, iDef.BestIpAddress(), cmds, *argVerbosity)

				case CmdCopyPrivateKeys:
					cmds, err := deploy.NewCopyPrivateKeysCommands(iDef)
					if err != nil {
						log.Fatalf("cannot build commands to copy private keys: %s", err.Error())
					}
					logMsg, finalErr = deploy.ExecCommandsOnInstance(prjPair.Live.SshConfig, iDef.BestIpAddress(), cmds, *argVerbosity)

				case CmdSetupServices:
					logMsg, finalErr = deploy.ExecScriptsOnInstance(prj.SshConfig, iDef.BestIpAddress(), iDef.Service.Env, prjPair.ProjectFileDirPath, iDef.Service.Cmd.Setup, *argVerbosity)

				case CmdStartServices:
					logMsg, finalErr = deploy.ExecScriptsOnInstance(prj.SshConfig, iDef.BestIpAddress(), iDef.Service.Env, prjPair.ProjectFileDirPath, iDef.Service.Cmd.Start, *argVerbosity)

				case CmdStopServices:
					logMsg, finalErr = deploy.ExecScriptsOnInstance(prj.SshConfig, iDef.BestIpAddress(), iDef.Service.Env, prjPair.ProjectFileDirPath, iDef.Service.Cmd.Stop, *argVerbosity)

				default:
					log.Fatalf("unknown service command:" + os.Args[1])
				}

				logChan <- logMsg
				errChan <- finalErr
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
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, volNickname string) {
					logMsg, err := deploy.CreateVolume(prjPair, volNickname, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, volNickname)
			}

		case CmdDeleteVolumes:
			commonArgs.Parse(os.Args[2:])
			errorsExpected = len(prjPair.Live.Volumes)
			errChan = make(chan error, errorsExpected)
			for volNickname, _ := range prjPair.Live.Volumes {
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, volNickname string) {
					logMsg, err := deploy.DeleteVolume(prjPair, volNickname, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, volNickname)
			}

		case CmdAttachVolumes:
			nicknames, err := getNicknamesArg(commonArgs, "instances")
			if err != nil {
				log.Fatalf(err.Error())
			}

			instances, err := filterByNickname(nicknames, prjPair.Live.Instances, "instance")
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
					go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string, volNickname string) {
						logMsg, err := deploy.AttachVolume(prjPair, iNickname, volNickname, *argVerbosity)
						logChan <- logMsg
						errChan <- err
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				}
			}

		case CmdUploadFiles:
			nicknames, err := getNicknamesArg(commonArgs, "file groups to upload")
			if err != nil {
				log.Fatalf(err.Error())
			}

			fileGroups, err := filterByNickname(nicknames, prjPair.Live.FileGroupsUp, "file group to upload")
			if err != nil {
				log.Fatalf(err.Error())
			}

			// Walk through src locally and create file upload specs and after-file specs
			fileSpecs, afterSpecs, err := deploy.FileGroupUpDefsToSpecs(&prjPair.Live, fileGroups)
			if err != nil {
				log.Fatalf(err.Error())
			}

			errorsExpected = len(fileSpecs)
			errChan = make(chan error, len(fileSpecs))
			for _, fuSpec := range fileSpecs {
				sem <- 1
				go func(prj *deploy.Project, logChan chan deploy.LogMsg, errChan chan error, fuSpec *deploy.FileUploadSpec) {
					logMsg, err := deploy.UploadFileSftp(prj, fuSpec.IpAddress, fuSpec.Src, fuSpec.Dst, fuSpec.DirPermissions, fuSpec.FilePermissions, fuSpec.Owner, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(&prjPair.Live, logChan, errChan, fuSpec)
			}

			fileUpErr := waitForWorkers(errorsExpected, errChan, logChan)
			if fileUpErr > 0 {
				os.Exit(fileUpErr)
			}

			errorsExpected = len(afterSpecs)
			errChan = make(chan error, len(afterSpecs))
			for _, aSpec := range afterSpecs {
				sem <- 1
				go func(prj *deploy.Project, logChan chan deploy.LogMsg, errChan chan error, aSpec *deploy.AfterFileUploadSpec) {
					logMsg, err := deploy.ExecScriptsOnInstance(prj.SshConfig, aSpec.IpAddress, aSpec.Env, prjPair.ProjectFileDirPath, aSpec.Cmd, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(&prjPair.Live, logChan, errChan, aSpec)
			}

		case CmdDownloadFiles:
			nicknames, err := getNicknamesArg(commonArgs, "file groups to download")
			if err != nil {
				log.Fatalf(err.Error())
			}

			fileGroups, err := filterByNickname(nicknames, prjPair.Live.FileGroupsDown, "file group to download")
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
				go func(prj *deploy.Project, logChan chan deploy.LogMsg, errChan chan error, fdSpec *deploy.FileDownloadSpec) {
					logMsg, err := deploy.DownloadFileSftp(prj, fdSpec.IpAddress, fdSpec.Src, fdSpec.Dst, *argVerbosity)
					logChan <- logMsg
					errChan <- err
				}(&prjPair.Live, logChan, errChan, fdSpec)
			}

		default:
			log.Fatalf("unknown command:" + os.Args[1])
		}
	}

	finalCmdErr := waitForWorkers(errorsExpected, errChan, logChan)
	if finalCmdErr > 0 {
		os.Exit(finalCmdErr)
	}

	// Save updated project template, it may have some new ids and timestamps
	if err = prjPair.Template.SaveProject(fullPrjPath); err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Printf("%s succeeded, elapsed %.3fs\n", os.Args[1], time.Since(cmdStartTs).Seconds())
}
