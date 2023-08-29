package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/deploy"
)

const (
	CmdCreateFloatingIp     string = "create_floating_ip"
	CmdDeleteFloatingIp     string = "delete_floating_ip"
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
	CmdInstallServices      string = "install_services"
	CmdConfigServices       string = "config_services"
	CmdStartServices        string = "start_services"
	CmdStopServices         string = "stop_services"
	CmdCreateInstanceUsers  string = "create_instance_users"
	CmdCopyPrivateKeys      string = "copy_private_keys"
	CmdPingInstances        string = "ping_instances"
	CmdBuildArtifacts       string = "build_artifacts"
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
	if len(os.Args[2]) == 0 {
		return "", fmt.Errorf("bad arg, expected comma-separated list of %s or '*'", entityName)
	}
	return os.Args[2], nil
}

func filterByNickname[GenericDef deploy.FileGroupUpDef | deploy.FileGroupDownDef | deploy.InstanceDef](nicknames string, sourceMap map[string]*GenericDef, entityName string) (map[string]*GenericDef, error) {
	var defMap map[string]*GenericDef
	rawNicknames := strings.Split(nicknames, ",")
	defMap = map[string]*GenericDef{}
	for _, rawNickname := range rawNicknames {
		if strings.Contains(rawNickname, "*") {
			matchFound := false
			reNickname := regexp.MustCompile("^" + strings.ReplaceAll(rawNickname, "*", "[a-zA-Z0-9]*") + "$")
			for fgNickname, fgDef := range sourceMap {
				if reNickname.MatchString(fgNickname) {
					matchFound = true
					defMap[fgNickname] = fgDef
				}
			}
			if !matchFound {
				return nil, fmt.Errorf("no match found for %s '%s', available definitions: %s", entityName, rawNickname, reflect.ValueOf(sourceMap).MapKeys())
			}
		} else {
			fgDef, ok := sourceMap[rawNickname]
			if !ok {
				return nil, fmt.Errorf("definition for %s '%s' not found, available definitions: %s", entityName, rawNickname, reflect.ValueOf(sourceMap).MapKeys())
			}
			defMap[rawNickname] = fgDef
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
  %s <comma-separated list of instances to create volumes on, or 'all'>
  %s <comma-separated list of instances to attach volumes on, or 'all'>
  %s <comma-separated list of instances to delete volumes on, or 'all'>
  %s <comma-separated list of instances to create, or 'all'>
  %s <comma-separated list of instances to delete, or 'all'>
  %s <comma-separated list of instances to ping, or 'all'>
  %s <comma-separated list of instances to create users on, or 'all'>
  %s <comma-separated list of instances to copy private keys to, or 'all'>
  %s <comma-separated list of upload file groups, or 'all'>
  %s <comma-separated list of download file groups, or 'all'>  
  %s <comma-separated list of instances to install services on, or 'all'>
  %s <comma-separated list of instances to config services on, or 'all'>
  %s <comma-separated list of instances to start services on, or 'all'>
  %s <comma-separated list of instances to stop services on, or 'all'>
`,
		CmdCreateFloatingIp,
		CmdDeleteFloatingIp,
		CmdCreateSecurityGroups,
		CmdDeleteSecurityGroups,
		CmdCreateNetworking,
		CmdDeleteNetworking,

		CmdCreateVolumes,
		CmdAttachVolumes,
		CmdDeleteVolumes,

		CmdCreateInstances,
		CmdDeleteInstances,
		CmdPingInstances,

		CmdCreateInstanceUsers,
		CmdCopyPrivateKeys,

		CmdUploadFiles,
		CmdDownloadFiles,

		CmdInstallServices,
		CmdConfigServices,
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
	argPrjFile := commonArgs.String("prj", "capideploy.json", "Capideploy project file path")

	if len(os.Args) <= 1 {
		usage(commonArgs)
		os.Exit(1)
	}

	cmdStartTs := time.Now()

	throttle := time.Tick(time.Second) // One call per second, to avoid error 429 on openstack calls
	const MaxWorkerThreads int = 10
	var logChan = make(chan deploy.LogMsg, MaxWorkerThreads*5)
	var sem = make(chan int, MaxWorkerThreads)
	var errChan chan error
	errorsExpected := 1
	var prjPair *deploy.ProjectPair
	var fullPrjPath string
	var prjErr error

	singleThreadCommands := map[string]SingleThreadCmdHandler{
		CmdCreateFloatingIp:     nil,
		CmdDeleteFloatingIp:     nil,
		CmdCreateSecurityGroups: nil,
		CmdDeleteSecurityGroups: nil,
		CmdCreateNetworking:     nil,
		CmdDeleteNetworking:     nil,
		CmdBuildArtifacts:       deploy.BuildArtifacts,
	}

	if _, ok := singleThreadCommands[os.Args[1]]; ok {
		commonArgs.Parse(os.Args[2:])
	} else {
		commonArgs.Parse(os.Args[3:])
	}

	prjPair, fullPrjPath, prjErr = deploy.LoadProject(*argPrjFile)
	if prjErr != nil {
		log.Fatalf(prjErr.Error())
	}

	deployProvider, deployProviderErr := deploy.DeployProviderFactory(prjPair.Template.DeployProviderName)
	if deployProviderErr != nil {
		log.Fatalf(deployProviderErr.Error())
	}
	singleThreadCommands[CmdCreateFloatingIp] = deployProvider.CreateFloatingIp
	singleThreadCommands[CmdDeleteFloatingIp] = deployProvider.DeleteFloatingIp
	singleThreadCommands[CmdCreateSecurityGroups] = deployProvider.CreateSecurityGroups
	singleThreadCommands[CmdDeleteSecurityGroups] = deployProvider.DeleteSecurityGroups
	singleThreadCommands[CmdCreateNetworking] = deployProvider.CreateNetworking
	singleThreadCommands[CmdDeleteNetworking] = deployProvider.DeleteNetworking

	if cmdHandler, ok := singleThreadCommands[os.Args[1]]; ok {
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
			usedKeypairs := map[string]struct{}{}
			for _, instDef := range instances {
				usedFlavors[instDef.FlavorName] = ""
				usedImages[instDef.ImageName] = ""
				usedKeypairs[instDef.RootKeyName] = struct{}{}
			}
			logMsg, err := deployProvider.GetFlavorIds(prjPair, usedFlavors, *argVerbosity)
			logChan <- logMsg
			DumpLogChan(logChan)
			if err != nil {
				log.Fatalf(err.Error())
			}

			logMsg, err = deployProvider.GetImageIds(prjPair, usedImages, *argVerbosity)
			logChan <- logMsg
			DumpLogChan(logChan)
			if err != nil {
				log.Fatalf(err.Error())
			}

			logMsg, err = deployProvider.GetKeypairs(prjPair, usedKeypairs, *argVerbosity)
			logChan <- logMsg
			DumpLogChan(logChan)
			if err != nil {
				log.Fatalf(err.Error())
			}

			fmt.Printf("Creating instances, consider clearing known_hosts to avoid ssh complaints:\n")
			for _, i := range instances {
				fmt.Printf("ssh-keygen -f ~/.ssh/known_hosts -R %s;\n", i.BestIpAddress())
			}

			for iNickname := range instances {
				<-throttle
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string) {
					logMsg, err := deployProvider.CreateInstanceAndWaitForCompletion(
						prjPair,
						iNickname,
						usedFlavors[prjPair.Live.Instances[iNickname].FlavorName],
						usedImages[prjPair.Live.Instances[iNickname].ImageName],
						prjPair.Live.Instances[iNickname].AvailabilityZone,
						*argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		case CmdDeleteInstances:
			for iNickname := range instances {
				<-throttle
				sem <- 1
				go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string) {
					logMsg, err := deployProvider.DeleteInstance(prjPair, iNickname, *argVerbosity)
					logChan <- logMsg
					errChan <- err
					<-sem
				}(prjPair, logChan, errChan, iNickname)
			}
		default:
			log.Fatalf("unknown create/delete instance command:" + os.Args[1])
		}
	} else if os.Args[1] == CmdPingInstances ||
		os.Args[1] == CmdCreateInstanceUsers ||
		os.Args[1] == CmdCopyPrivateKeys ||
		os.Args[1] == CmdInstallServices ||
		os.Args[1] == CmdConfigServices ||
		os.Args[1] == CmdStartServices ||
		os.Args[1] == CmdStopServices {
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
			<-throttle
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

				case CmdInstallServices:
					logMsg, finalErr = deploy.ExecScriptsOnInstance(prj.SshConfig, iDef.BestIpAddress(), iDef.Service.Env, prjPair.ProjectFileDirPath, iDef.Service.Cmd.Install, *argVerbosity)

				case CmdConfigServices:
					logMsg, finalErr = deploy.ExecScriptsOnInstance(prj.SshConfig, iDef.BestIpAddress(), iDef.Service.Env, prjPair.ProjectFileDirPath, iDef.Service.Cmd.Config, *argVerbosity)

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

	} else if os.Args[1] == CmdCreateVolumes || os.Args[1] == CmdAttachVolumes || os.Args[1] == CmdDeleteVolumes {
		nicknames, err := getNicknamesArg(commonArgs, "instances")
		if err != nil {
			log.Fatalf(err.Error())
		}

		instances, err := filterByNickname(nicknames, prjPair.Live.Instances, "instance")
		if err != nil {
			log.Fatalf(err.Error())
		}

		volCount := 0
		for _, iDef := range instances {
			volCount += len(iDef.Volumes)
		}
		if volCount == 0 {
			fmt.Printf("No volumes to create/attach/delete")
			os.Exit(0)
		}
		errorsExpected = volCount
		errChan = make(chan error, volCount)
		for iNickname, iDef := range instances {
			for volNickname := range iDef.Volumes {
				<-throttle
				sem <- 1
				switch os.Args[1] {
				case CmdCreateVolumes:
					go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string, volNickname string) {
						logMsg, err := deployProvider.CreateVolume(prjPair, iNickname, volNickname, *argVerbosity)
						logChan <- logMsg
						errChan <- err
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				case CmdAttachVolumes:
					go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string, volNickname string) {
						logMsg, err := deployProvider.AttachVolume(prjPair, iNickname, volNickname, *argVerbosity)
						logChan <- logMsg
						errChan <- err
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				case CmdDeleteVolumes:
					go func(prjPair *deploy.ProjectPair, logChan chan deploy.LogMsg, errChan chan error, iNickname string, volNickname string) {
						logMsg, err := deployProvider.DeleteVolume(prjPair, iNickname, volNickname, *argVerbosity)
						logChan <- logMsg
						errChan <- err
						<-sem
					}(prjPair, logChan, errChan, iNickname, volNickname)
				default:
					log.Fatalf("unknown command:" + os.Args[1])
				}
			}
		}
	} else {
		switch os.Args[1] {
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
	if prjErr = prjPair.Template.SaveProject(fullPrjPath); prjErr != nil {
		log.Fatalf(prjErr.Error())
	}

	fmt.Printf("%s %sOK%s, elapsed %.3fs\n", os.Args[1], deploy.LogColorGreen, deploy.LogColorReset, time.Since(cmdStartTs).Seconds())
}
