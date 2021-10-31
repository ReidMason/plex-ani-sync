export function joinUrl(...paths: Array<string>): string {
    // Remove trailing and leading slashes
    paths = paths.map((path) => path.replace(/^\/*|\/*$/gm, ""))

    return paths.join("/")
}

export function createRequestUrl(endpoint: string): string {
    const baseUrl: string = process.env.REACT_APP_API_BASE_URL!;

    return joinUrl(baseUrl, endpoint)
}