// Hand-built approximations of the Jira and Azure DevOps marks for the
// tracker-kind picker. Not pixel-exact official brand assets — swap for the
// real SVGs if precise branding ever matters (these are "coming soon" cards).

export function JiraIcon({className}: { className?: string }) {
    return (
        <svg viewBox="0 0 24 24" fill="none" className={className} xmlns="http://www.w3.org/2000/svg">
            <path
                d="M12 2 3 11a2.6 2.6 0 0 0 0 3.7L12 23l3.9-3.9-6.1-6.1a1 1 0 0 1 0-1.4L12 9.2 15.9 13a2.6 2.6 0 0 0 3.7 0L21 11.6a2.6 2.6 0 0 0 0-3.7L12 2Z"
                fill="#2684FF"
            />
            <path
                d="M12 9.2 8.1 13a2.6 2.6 0 0 0 0 3.7L12 20.6l3.9-3.9-3.9-3.8a1 1 0 0 1 0-1.4L12 9.2Z"
                fill="#0052CC"
            />
        </svg>
    )
}

export function AzureDevOpsIcon({className}: { className?: string }) {
    return (
        <svg viewBox="0 0 24 24" fill="none" className={className} xmlns="http://www.w3.org/2000/svg">
            <path d="M22 6.3 16.9 2v3.1L8.4 8.6v-3L2 9.7v8.6l5 2v-3.2l1.9.8L22 13.5V6.3Z" fill="#0078D4" />
            <path
                d="M22 6.3v7.2l-13.1 5.6-2-.8v3.2l-5-2V9.7l4.5-4v9.4l4 1.7 9.6-8.4L22 6.3Z"
                fill="#0078D4"
                fillOpacity="0.6"
            />
        </svg>
    )
}
