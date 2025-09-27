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
    sh "docker build --build-arg APP_NAME=${appName} -t ${appName}:${tag} ."
}

def runTests() {
    sh 'go test ./... -v'
}

def archiveArtifacts(appName, sha) {
    def tag = (sha == null || sha.trim() == '') ? 'latest' : sha
    if (!sha?.trim()) {
        echo "⚠️ FATAL: Commit SHA is empty, tagging as 'latest'"
    }
    echo "PUSH IMAGE TO REGISTRY..."
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
        to: getEmailRecipients()
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
        to: getEmailRecipients()
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