#!groovy
@Library('github.com/cloudogu/ces-build-lib@3.0.0')
import com.cloudogu.ces.cesbuildlib.*

git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)

repositoryName = "k8s-ces-gateway"
productionReleaseBranch = "main"

registryNamespace = "k8s"
registryUrl = "registry.cloudogu.com"

goVersion = "1.24"
helmTargetDir = "target/k8s"
helmChartDir = "${helmTargetDir}/helm"

node('docker') {
    timestamps {
        catchError {
            timeout(activity: false, time: 60, unit: 'MINUTES') {
                stage('Checkout') {
                    checkout scm
                    make 'clean'
                }

                new Docker(this)
                        .image("golang:${goVersion}")
                        .mountJenkinsUser()
                        .inside("--volume ${WORKSPACE}:/${repositoryName} -w /${repositoryName}")
                                {
                                    stage('Generate k8s Resources') {
                                        make 'helm-update-dependencies'
                                        make 'helm-generate'
                                        archiveArtifacts "${helmTargetDir}/**/*"
                                    }

                                    stage("Lint helm") {
                                        make 'helm-lint'
                                    }
                                }

                K3d k3d = new K3d(this, "${WORKSPACE}", "${WORKSPACE}/k3d", env.PATH)

                try {
                    stage('Set up k3d cluster') {
                        k3d.startK3d()
                    }

                    stage('Deploy k8s-ces-gateway') {
                        k3d.helm("install ${repositoryName} ${helmChartDir}")
                    }

                    stage('Test k8s-ces-gateway') {
                        // Sleep because it takes time for the controller to create the resource. Without it would end up
                        // in error "no matching resource found when run the wait command"
                        sleep(20)
                        k3d.kubectl("wait --for=condition=ready pod -l app.kubernetes.io/instance=k8s-ces-gateway-default --timeout=300s")
			// Make sure that TLS v1.2 is disabled and TLS v1.3 is available
			sh '''
			# only gets the first serverlb container name, if there are multiple running
			container_name=$(docker ps --filter "name=k3d-citest-.*-serverlb" --format "{{.Names}}")
			ip_and_port=$(docker port $container_name 6443)
			# Check for TLS v1.2 not being available
			openssl s_client -connect $ip_and_port -tls1_2 -brief >/dev/null 2>&1
			TLS1_2_available=$?
			if [ "$TLS1_2_available" -eq 0 ]; then
			  echo "ERROR: TLS 1.2 is available, but it should be disabled!"
			  exit 1
			fi
			# Check for TLS v1.3 being available
			openssl s_client -connect $ip_and_port -tls1_3 -brief >/dev/null 2>&1
			TLS1_3_available=$?
			if [ "$TLS1_3_available" -ne 0 ]; then
			  echo "ERROR: TLS 1.3 is NOT available!"
			  exit 1
			fi
			'''
                    }
                } catch(Exception e) {
                    k3d.collectAndArchiveLogs()
                    throw e as java.lang.Throwable
                } finally {
                    stage('Remove k3d cluster') {
                        k3d.deleteK3d()
                    }
                }
            }
        }

        stageAutomaticRelease()
    }
}

void stageAutomaticRelease() {
    if (gitflow.isReleaseBranch()) {
        Makefile makefile = new Makefile(this)
        String releaseVersion = makefile.getVersion()
        String changelogVersion = git.getSimpleBranchName()

        stage('Push Helm chart to Harbor') {
            new Docker(this)
                    .image("golang:${goVersion}")
                    .mountJenkinsUser()
                    .inside("--volume ${WORKSPACE}:/${repositoryName} -w /${repositoryName}")
                            {
                                make 'helm-package'
                                archiveArtifacts "${helmTargetDir}/**/*"

                                withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'harborhelmchartpush', usernameVariable: 'HARBOR_USERNAME', passwordVariable: 'HARBOR_PASSWORD']]) {
                                    sh ".bin/helm registry login ${registryUrl} --username '${HARBOR_USERNAME}' --password '${HARBOR_PASSWORD}'"
                                    sh ".bin/helm push ${helmChartDir}/${repositoryName}-${releaseVersion}.tgz oci://${registryUrl}/${registryNamespace}"
                                }
                            }
        }

        stage('Finish Release') {
            gitflow.finishRelease(changelogVersion, productionReleaseBranch)
        }

        stage('Add Github-Release') {
            releaseId = github.createReleaseWithChangelog(changelogVersion, changelog, productionReleaseBranch)
        }
    }
}

void make(String makeArgs) {
    sh "make ${makeArgs}"
}
