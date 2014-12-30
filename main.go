package main

import (
	"./mail"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libgit2/git2go"
	"os"
	"time"
)

type Configuration struct {
	User         string
	Password     string
	MailUser     string
	MailPassword string
	MailServer   string
	MailPort     int
	MailTo       string
}

func slice(args ...interface{}) []interface{} {
	return args
}

func main() {
	cloneTextSecureHellmy()
	rep, _ := git.OpenRepository(os.Getenv("HOME") + "/TS-hellmy")

	rep.CreateRemote("WhisperSystems", "https://github.com/WhisperSystems/TextSecure.git")
	remoteWS, _ := rep.LoadRemote("WhisperSystems")
	var s []string
	remoteWS.Fetch(s, nil, "")

	remoteOrigin, _ := rep.LoadRemote("origin")
	remoteOrigin.Fetch(s, nil, "")

	// Switch to my_version
	if err := checkoutBranch(rep, "origin/my_version"); err != nil {
		fmt.Println("Some error during checkout....", err)
		return
	}

	// Delete my_version?

	// Create my_version with master (force?!)

	// Merge WS/master
	merge(rep, "WhisperSystems/master")

	// Merge branch origin/msg_details_icon
	merge(rep, "origin/msg_details_icon")

	// push origin/my_version
	pushOrigin(remoteOrigin)

}

func cloneTextSecureHellmy() {
	cloneOptions := new(git.CloneOptions)

	homeEnv := os.Getenv("HOME")
	fmt.Println(git.Clone("https://github.com/Hellmy/TextSecure.git", homeEnv+"/TS-hellmy", cloneOptions))
}

func checkoutBranch(rep *git.Repository, branchName string) error {
	signature := new(git.Signature)

	branchReference, lookupErr := rep.LookupReference("refs/remotes/" + branchName)

	if lookupErr != nil {
		return lookupErr
	}
	targetCommit, _ := rep.LookupCommit(branchReference.Target())
	branch, _ := rep.CreateBranch(branchName, targetCommit, true, signature, "")
	fmt.Println("Created Branch: ", branch)

	checkoutHeadOpts := new(git.CheckoutOpts)
	checkoutHeadOpts.Strategy = git.CheckoutForce

	branchTree, _ := rep.LookupTree(targetCommit.TreeId())
	fmt.Println("checkoutTree Error: ", rep.CheckoutTree(branchTree, checkoutHeadOpts))

	fmt.Println("setHead Error: ", rep.SetHead("refs/heads/"+branchName, signature, ""))

	return nil

}

func merge(rep *git.Repository, branchName string) {
	if branch, err := mergeBranch(rep, branchName); err != nil {
		fmt.Println(branchName+" mergeBranch() error: ", err)
		mailMessage(err.Error())
	} else {
		if err := commitMergedBranch(rep, branch); err != nil {
			fmt.Println(branchName+" commmit error: ", err)
		}
	}
}

func mergeBranch(rep *git.Repository, branchName string) (*git.Branch, error) {
	ref, lookupError := rep.LookupReference("refs/remotes/" + branchName)
	if lookupError != nil {
		return nil, lookupError
	}
	mergeHead, mergeError := rep.MergeHeadFromRef(ref)

	if mergeError != nil {
		return nil, mergeError
	}

	mergeAnalysis, _, mergeAnalysisError := rep.MergeAnalysis([]*git.MergeHead{mergeHead})
	fmt.Println("Merge Analysis: ", mergeAnalysis, "Merge analysis error: ", mergeAnalysisError)

	if mergeAnalysisError != nil {
		return nil, mergeAnalysisError
	}
	if (mergeAnalysis & git.MergeAnalysisUnborn) == git.MergeAnalysisUnborn {
		return nil, errors.New("Tried to merge an 'unborn' commit.")
	} else if (mergeAnalysis & git.MergeAnalysisFastForward) == git.MergeAnalysisFastForward {
		fmt.Println("FastForward Merge")
		if err := rep.Merge([]*git.MergeHead{mergeHead}, nil, nil); err != nil {
			return nil, err
		}
	} else if (mergeAnalysis & git.MergeAnalysisUpToDate) == git.MergeAnalysisUpToDate {
		fmt.Println("Up to date merge")

		opts := new(git.CheckoutOpts)
		opts.Strategy = git.CheckoutForce
		rep.CheckoutHead(opts)

		return nil, nil
	} else if (mergeAnalysis & git.MergeAnalysisNormal) == git.MergeAnalysisNormal {
		fmt.Println("Normal Merge")
		if err := rep.Merge([]*git.MergeHead{mergeHead}, nil, nil); err != nil {
			return nil, err
		}
	} else if (mergeAnalysis & git.MergeAnalysisNone) == git.MergeAnalysisNone {
		return nil, errors.New("No merge is possible")
	}

	return ref.Branch(), mergeError
}

func mailMessage(body string) {
	config, err := readConfig()
	if err != nil {
		return
	}
	emailUser := &mail.EmailUser{Username: config.MailUser, Password: config.MailPassword, EmailServer: config.MailServer, Port: config.MailPort}
	smtpData := &mail.SmtpTemplateData{From: config.MailUser, To: config.MailTo, Subject: "git-merger Error", Body: body}
	mail.Mail(*emailUser, *smtpData)
}

func commitMergedBranch(rep *git.Repository, branch *git.Branch) error {
	if branch == nil {
		return nil
	}
	signature := git.Signature{Name: "Manuel", Email: "manuel@zeunerds.de", When: time.Now()}
	headReference, _ := rep.Head()

	fmt.Println("Head Name: ", headReference.Name())

	headIndex, _ := rep.Index()
	if headIndex.HasConflicts() {
		return errors.New("The Index has some conflicts")
	}
	headWriteOid, _ := headIndex.WriteTree()
	headTree, _ := rep.LookupTree(headWriteOid)
	fmt.Println("head Index Tree: ", headTree)

	currentTip, _ := rep.LookupCommit(headReference.Target())
	branchTip, _ := rep.LookupCommit(branch.Target())
	branchName, _ := branch.Name()

	commitId, e := rep.CreateCommit("HEAD", &signature, &signature, "merged  "+branchName, headTree, currentTip, branchTip)

	fmt.Println("commitId: ", commitId, "Error: ", e)
	return nil
}

func pushOrigin(remoteOrigin *git.Remote) {

	configuration, err := readConfig()
	if err != nil {
		return
	}

	fmt.Println("Remote url: ", remoteOrigin.Url())

	remoteCallback := new(git.RemoteCallbacks)
	remoteCallback.CredentialsCallback = func(url string, username_from_url string, allowed_types git.CredType) (git.ErrorCode, *git.Cred) {
		_, cred := git.NewCredUserpassPlaintext(configuration.User, configuration.Password)
		return git.ErrOk, &cred
	}
	remoteOrigin.SetCallbacks(remoteCallback)

	//fmt.Println("Connect Push Error: ", remoteOrigin.ConnectPush())

	pushCallbacks := new(git.PushCallbacks)
	packbuilderProgress := new(git.PackbuilderProgressCallback)
	*packbuilderProgress = func(stage int, current uint, total uint) int {
		fmt.Println("Packbuilder - stage:", stage, "current: ", current, "total: ", total)

		return 0
	}
	pushCallbacks.PackbuilderProgress = packbuilderProgress

	pushTransferProgressCallback := new(git.PushTransferProgressCallback)
	*pushTransferProgressCallback = func(current uint, total uint, bytes uint) int {

		fmt.Println("TranferProgress - current: ", current, "total: ", total, "bytes: ", bytes)
		return 0
	}
	pushCallbacks.TransferProgress = pushTransferProgressCallback

	pushOrigin, _ := remoteOrigin.NewPush()
	fmt.Println("Add Refspec Error: ", pushOrigin.AddRefspec("refs/heads/origin/my_version:refs/heads/my_version"))
	pushOrigin.SetCallbacks(*pushCallbacks)
	fmt.Println("Finish Push Error: ", pushOrigin.Finish())
	fmt.Println("Unpack Push Ok: ", pushOrigin.UnpackOk())

	fmt.Println("StatusForEach Error: ", pushOrigin.StatusForeach(func(ref string, msg string) int {
		fmt.Println("StatusForEach - ref: ", ref, "msg: ", msg)

		return 0
	}))
}

func readConfig() (Configuration, error) {
	file, _ := os.Open(os.Getenv("HOME") + "/git-merger.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("No configuration file found")
		return configuration, err
	}
	return configuration, nil
}
