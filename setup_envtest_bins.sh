#!/bin/sh

#  Copyright 2020 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# This file will be  fetched as: curl -L https://git.io/getLatestKubebuilder | sh -
# so it should be pure bourne shell, not bash (and not reference other scripts)

set -eu

# To use envtest is required to have etcd, kube-apiserver and kubetcl binaries installed locally.
# This script will create the directory testbin and perform this setup for linux or mac os x envs in

#Validates the user has passed two command line arguments and prints usage if not
validateArgs() {
  #Validate Correct Number of Arguments
  if [ "$#" -ne 2 ]; then
    echo ""
    echo "    USAGE:"
    echo "    setup_testenv_bin.sh k8s_version etcd_version"
    echo "        k8s_version   string |  represents a valid version of k8s, such as v1.18.6"
    echo "        etcd_version  string |  represents a valid version of etcd, such as v3.4.3"
    echo ""
    exit 1
  fi
}

#Sets the global script variables for use throughout other script functions
setEnv() {
  # Kubernetes version e.g v1.18.2
  K8S_VER=$1
  # ETCD version e.g v3.4.3
  ETCD_VER=$2

  #Operating System
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  #CPU Architecture
  ARCH=$(uname -m | sed 's/x86_64/amd64/')

  #Directory where we will place etcd and k8s executable files
  TESTBIN_DIR=testbin

  #Sets the archive format depending upon the operating system type.  tar.gz for linux.  zip for macos
  EXT="tar.gz"
  if [ ${OS} = "darwin" ]; then
       EXT="zip"
  fi

  #Builds ETCD distribution file name
  ETCD_DIST=etcd-${ETCD_VER}-${OS}-${ARCH}
  ETCD_DIST_FILE=${ETCD_DIST}.${EXT}

  #Builds K8S distribution file name
  K8S_DIST=kubernetes-server-${OS}-${ARCH}
  K8S_DIST_FILE=${K8S_DIST}.tar.gz

  #PRINTING VARIABLES
  #echo "K8S_VER ${K8S_VER}"
  #echo "ETCD_VER ${ETCD_VER}"
  #echo "${OS}"
  #echo "${ARCH}"
  #echo "ETCD_DIST ${ETCD_DIST}"
  #echo "EXT ${EXT}"
  #echo "ETCD_DIST_FILE ${ETCD_DIST_FILE}"
  #echo "K8S_DIST ${K8S_DIST}
  #echo "K8S_DIST_FILE ${K8S_DIST_FILE}
  #echo "$TESTBIN_DIR ${TESTBIN_DIR}"
}

#Generic download function used to obtain ETCD and K8S distribution archives
download() {
  local DOWNLOAD_URL=$1
  local FILE=$2
  local EXTRACT_DIR=$3

  mkdir -p $EXTRACT_DIR

  #Attempt to download the archive from the url, place the archive into the extraction directory and exit on failure
  if wget -q ${DOWNLOAD_URL}; then
    echo "Download of ${DOWNLOAD_URL} completed."
    mv ${FILE} ${EXTRACT_DIR}
  else
      echo "${DOWNLOAD_URL} does not exist.  Exiting."
      exit 1
  fi
}

#Downloads ETCD from github in order to extract the executable and removes the tarball upon completion
downloadETCD() {
  local URL=https://github.com/etcd-io/etcd/releases/download/${ETCD_VER}/${ETCD_DIST_FILE}
  download ${URL} ${ETCD_DIST_FILE} ${TESTBIN_DIR} 
  tar -zxf ${TESTBIN_DIR}/${ETCD_DIST_FILE} -C ${TESTBIN_DIR} --strip-components=1 ${ETCD_DIST}/etcd ${ETCD_DIST}/etcdctl
  rm ${TESTBIN_DIR}/${ETCD_DIST_FILE}
}

#Check if etcd executable has been downloaded already.  Downloads if not present.
getETCD() {
  if [ ! -e ${TESTBIN_DIR}/etcd ]; then
    downloadETCD
  fi
}

#Downloads K8S from kubernetes website in order to extract executables and removes the tarball upon completion
downloadK8S(){
  local URL=https://dl.k8s.io/${K8S_VER}/${K8S_DIST_FILE}
  download ${URL} ${K8S_DIST_FILE} ${TESTBIN_DIR}
  tar -zxf ${TESTBIN_DIR}/${K8S_DIST_FILE} -C ${TESTBIN_DIR} --strip-components=3 kubernetes/server/bin/kube-apiserver kubernetes/server/bin/kubectl
  rm ${TESTBIN_DIR}/${K8S_DIST_FILE}
}

#On a Linux Operating System
#Check if k8s executables have been downloaded already.  Downloads if not present.
getK8s4Linux() {
  if [ ! -e ${TESTBIN_DIR}/kube-apiserver ] || [ ! -e ${TESTBIN_DIR}/kubectl ]; then
    downloadK8S
  fi
}

#On a MacOS Operating System
#Check if k8s executables have been downloaded already.  Downloads if not present.
getK8s4Mac() {
  # kubernetes do not provide the kubernetes-server for darwin,
  # In this way, to have the kube-apiserver is required to build it locally
  # if the project is cloned locally already do nothing
  if [ ! -d $GOPATH/src/k8s.io/kubernetes ]; then
  git clone https://github.com/kubernetes/kubernetes $GOPATH/src/k8s.io/kubernetes --depth=1 -b ${K8S_VER}
  fi

  # if the kube-apiserve is built already then, just copy it
  if [ ! -f $GOPATH/src/k8s.io/kubernetes/_output/local/bin/darwin/amd64/kube-apiserver ]; then
  DIR=$(pwd)
  cd $GOPATH/src/k8s.io/kubernetes
  # Build for linux first otherwise it won't work for darwin - :(
  export KUBE_BUILD_PLATFORMS="linux/amd64"
  make WHAT=cmd/kube-apiserver
  export KUBE_BUILD_PLATFORMS="darwin/amd64"
  make WHAT=cmd/kube-apiserver
  cd ${DIR}
  fi
  cp $GOPATH/src/k8s.io/kubernetes/_output/local/bin/darwin/amd64/kube-apiserver $TESTBIN_DIR/

  # setup kubectl binary
  curl -LO https://storage.googleapis.com/kubernetes-release/release/${K8S_VER}/bin/darwin/amd64/kubectl
  chmod +x kubectl
  mv kubectl $TESTBIN_DIR/

  # allow run the tests without the Mac OS Firewall popup shows for each execution
  codesign --deep --force --verbose --sign - ./${TESTBIN_DIR}/kube-apiserver
}

#Ensures K8S is available in the test environment.  The process differs for MacOS and Linux.
#Branching according to the operating system type occurs here.
getK8S() {
  # install kube-apiserver and kubetcl binaries
  if [ ${OS} = "darwin" ]
  then
    getK8s4MAC
  else
    getK8s4Linux
  fi
}

#Drives test environment setup
setup_testenv_bin() {
  # Do nothing if the $TESTBIN_DIR directory exist already.
  if [ ! -d $TESTBIN_DIR ]; then
    mkdir -p $TESTBIN_DIR
    getETCD
    getK8S 
  fi

  #Export Test Environment Variables
  export PATH=/$TESTBIN_DIR:$PATH
  export TEST_ASSET_KUBECTL=/$TESTBIN_DIR/kubectl
  export TEST_ASSET_KUBE_APISERVER=/$TESTBIN_DIR/kube-apiserver
  export TEST_ASSET_ETCD=/$TESTBIN_DIR/etcd
}

#Kickoff the test environment setup
validateArgs "$@"
setEnv "$@"
setup_testenv_bin
