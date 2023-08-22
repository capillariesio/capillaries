package deploy

func (*AwsDeployProvider) GetFlavorIds(prjPair *ProjectPair, flavorMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetFlavorIds", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) GetImageIds(prjPair *ProjectPair, imageMap map[string]string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetImageIds", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) GetKeypairs(prjPair *ProjectPair, keypairMap map[string]struct{}, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.GetKeypairs", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) CreateInstanceAndWaitForCompletion(prjPair *ProjectPair, iNickname string, flavorId string, imageId string, availabilityZone string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.CreateInstanceAndWaitForCompletion", isVerbose)
	return lb.Complete(nil)
}
func (*AwsDeployProvider) DeleteInstance(prjPair *ProjectPair, iNickname string, isVerbose bool) (LogMsg, error) {
	lb := NewLogBuilder("aws.DeleteInstance", isVerbose)
	return lb.Complete(nil)
}
