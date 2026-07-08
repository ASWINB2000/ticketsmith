// Hand-built approximations of brand marks (Jira, Azure DevOps, GitHub). Not
// pixel-exact official brand assets — swap for the real SVGs if precise
// branding ever matters.

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

export function GitHubIcon({className}: { className?: string }) {
    return (
        <svg viewBox="0 0 24 24" fill="currentColor" className={className} xmlns="http://www.w3.org/2000/svg">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.071 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.833.09-.647.35-1.088.636-1.339-2.221-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.269 2.75 1.026A9.548 9.548 0 0 1 12 6.844c.85.004 1.705.115 2.504.337 1.909-1.295 2.747-1.026 2.747-1.026.546 1.378.203 2.397.1 2.65.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.31.678.92.678 1.855 0 1.338-.012 2.419-.012 2.749 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2Z"
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
