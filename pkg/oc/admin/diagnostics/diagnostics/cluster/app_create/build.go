package app_create

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kapi "k8s.io/kubernetes/pkg/apis/core"

	build "github.com/openshift/origin/pkg/build/apis/build"
	buildclientinternal "github.com/openshift/origin/pkg/build/client/internalversion"
	image "github.com/openshift/origin/pkg/image/apis/image"
)

const base64Binary string = ""

func (d *AppCreate) createAndTestBuild() {
	result := &d.result.Build
	result.BeginTime = jsonTime(time.Now())
	defer recordTrial(result)
	if !d.createBuild() {
		return
	}
}

// create the build for the app
func (d *AppCreate) createBuild() bool {
	// check whether the python s2i builder even exists before we begin
	if _, err := d.ImageStreamClient.ImageStreams("openshift").Get("python", metav1.GetOptions{}); errors.IsNotFound(err) {
		d.out.Warn("DCluAC055", nil, "No 'python' ImageStream in project 'openshift' to build with; skipping build")
		return false
	} else if err != nil {
		d.out.Error("DCluAC056", err, fmt.Sprintf("Error trying to detect ImageStream openshift/python:\n%v\nBuild will be skipped.", err))
		return false
	}

	defer recordTime(&d.result.Build.CreatedTime)
	objectMeta := metav1.ObjectMeta{Namespace: d.project, Name: d.appName, Labels: d.label}
	// create the imagestream for our build to land in
	is := &image.ImageStream{ObjectMeta: objectMeta}
	if _, err := d.ImageStreamClient.ImageStreams(d.project).Create(is); err != nil {
		d.out.Error("DCluAC057", err, fmt.Sprintf("Error trying to create the '%s' ImageStream to build into:\n%v\nBuild will be skipped.", d.appName, err))
		return false
	}

	// create the buildconfig to run our build
	bcclient := d.BuildClient.Build().BuildConfigs(d.project)
	config := &build.BuildConfig{
		ObjectMeta: objectMeta,
		Spec: build.BuildConfigSpec{
			CommonSpec: build.CommonSpec{
				Strategy: build.BuildStrategy{
					SourceStrategy: &build.SourceBuildStrategy{
						From: kapi.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      "python:latest",
							Namespace: "openshift",
						},
					},
				},
				Output: build.BuildOutput{
					To: &kapi.ObjectReference{
						Kind:      "ImageStream",
						Namespace: d.project,
						Name:      d.appName,
					},
				},
			},
		},
	}
	if _, err := bcclient.Create(config); err != nil {
		d.out.Error("DCluAC058", err, fmt.Sprintf("%s: Creating build config '%s' failed:\n%v", now(), d.appName, err))
		return false
	}

	binaryClient := buildclientinternal.NewBuildInstantiateBinaryClient(d.BuildClient.Build().RESTClient(), d.project)
	bropts := &build.BinaryBuildRequestOptions{
		ObjectMeta: objectMeta,
		Message:    "AppCreate diagnostic",
	}
	binaryReader, err := base64.StdEncoding.DecodeString(base64Binary)
	if err != nil {
		panic(err)
	}
	build, err := binaryClient.InstantiateBinary(d.appName, bropts, bytes.NewReader(binaryReader))
	if err != nil {
		d.out.Error("DCluAC059", err, fmt.Sprintf("%s: Instantiating a build from '%s' failed:\n%v", now(), d.appName, err))
		return false
	}
	build.GetName()

	return true
}
