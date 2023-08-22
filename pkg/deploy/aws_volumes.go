package deploy

func (*AwsDeployProvider) CreateVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.CreateVolume", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) AttachVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.AttachVolume", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) DeleteVolume(prjPair *ProjectPair, iNickname string, volNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteVolume", isVerbose)
	return lb.Complete(nil)
}
