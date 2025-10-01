//TODO: =========== Initialize functions ===========
def getParameters() {
    return [
        choice(name: 'ENVIRONMENT', choices: ['STAG', 'PROD'], description: 'Select environment for deployment'),
        booleanParam(name: 'RUN_TESTS', defaultValue: false, description: 'Run tests after build')
    ]
}

def getEmailRecipients() {
    return "trangiangkhanh04@gmail.com"
}

//TODO: =========== Build functions ===========
def checkoutSSH(url, branchRegex) {
    checkout([
        $class: 'GitSCM',
        branches: [[name: branchRegex]],
        doGenerateSubmoduleConfigurations: false,
        extensions: [],
        userRemoteConfigs: [[
            url: url,
            credentialsId: 'github-ssh'
        ]]
    ])
}

def buildDockerfile(appName, sha) {
    def tag = (sha == null || sha.trim() == '') ? 'latest' : sha
    if (!sha?.trim()) {
        echo "⚠️ FATAL: Commit SHA is empty, tagging as 'latest'"
    }

    def imagePrefix = "${appName}:${tag}"
    def imageName = "ghcr.io/sep490-project/core-backend/${appName}"

    // Remove any old images that start with the same short SHA
    sh """
        old_images=\$(docker images --format '{{.Repository}}:{{.Tag}}' | grep '^${appName}:${tag}' || true)
        if [ -n "\$old_images" ]; then
            echo "🗑 Removing old images with prefix ${imagePrefix}"
            docker rmi -f \$old_images || true
        fi
    """

    sh """
        docker build --build-arg APP_NAME=${appName} -t ${imageName}:${tag} .
    """
}


def runTests() {
    sh 'go test ./... -v'
}

def archiveArtifacts(appName, sha, branchName) {
    if (sha == null || sha.trim() == '') {
        error "FATAL: Commit SHA is required for archiving artifacts."
        return
    }
    
    def registry = "ghcr.io"
    def imageName = "ghcr.io/sep490-project/core-backend/${appName}"

    def sanitizedBranchName = branchName.replaceAll('/', '-')

    def imageWithShaTag = "${imageName}:${sha}"
    def imageWithBranchTag = "${imageName}:${sanitizedBranchName}"
    def imageWithLatestTag = "${imageName}:latest"

    withCredentials([usernamePassword(credentialsId: 'ghcr-access',
                                      usernameVariable: 'GH_USER',
                                      passwordVariable: 'GH_PAT')]) {
        sh """
          # Exit immediately if a command exits with a non-zero status.
          set -e

          echo "Logging in to Docker registry at ${registry}..."
          echo "\$GH_PAT" | docker login ${registry} -u "\$GH_USER" --password-stdin

          echo "Tagging image ${imageWithShaTag} with additional tags..."
          docker tag ${imageWithShaTag} ${imageWithBranchTag}
          docker tag ${imageWithShaTag} ${imageWithLatestTag}

          echo "Pushing tags to the registry..."
          docker push ${imageWithShaTag}
          docker push ${imageWithBranchTag}
          docker push ${imageWithLatestTag}
        """
    }
}

//TODO: =========== Notification functions ===========
def sendSuccessNotification() {
    emailext(
        subject: "✅ Build Success: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
        body: """
        <h2>Build thành công</h2>
        <p>Job: ${env.JOB_NAME}</p>
        <p>Build number: ${env.BUILD_NUMBER}</p>
        <p>Environment: ${params.ENVIRONMENT}</p>
        <p>Xem chi tiết tại: <a href="${env.BUILD_URL}">${env.BUILD_URL}</a></p>
        """,
        attachLog: true,
        to: getEmailRecipients(),
        mimeType: 'text/html'
    )
}

def sendFailureNotification() {
    emailext(
        subject: "❌ Build Failed: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
        body: """
        <h2>Build thất bại</h2>
        <p>Job: ${env.JOB_NAME}</p>
        <p>Build number: ${env.BUILD_NUMBER}</p>
        <p>Environment: ${params.ENVIRONMENT}</p>
        <p>Xem log tại: <a href="${env.BUILD_URL}console">${env.BUILD_URL}console</a></p>
        """,
        attachLog: true,
        to: getEmailRecipients(),
        mimeType: 'text/html'
    )
}

//TODO: =========== Utility functions ===========
def deployToEnvironment(environment, appName, sha) {
    switch(environment) {
        case 'STAG':
            echo "Deploying ${appName} to STAGING environment (RUN LOCALLY)"
            sh "docker run -d -p 8080:8080 --name ${appName}_${sha}_STAG ${appName}:${sha}"
            break
        case 'PROD':
            echo "Deploying ${appName} to PRODUCTION environment (NOT SUPPORTED YET)"
            break
        default:
            error "Unknown environment: ${environment}"
    }
}

return this
