
import { Api } from "../utils/api";

export const handleLogin = async (username: string, password: string): Promise<{success: boolean, error?: string}>  => {
    let response = null;
    try {
        response = await Api.post('/auth/authorize', {username, password}, false, true)
    } catch (error) {
        return {success: false, error: error instanceof Error ? error.message : 'Login failed'}
    }

    if (!response.code) {
        return {success: false, error: 'Invalid response from server'}

    }

    let tokenResponse = null;
    try {
        tokenResponse = await Api.post('/auth/token', {'grant_type': 'authorization_code', 'code': response.code}, false, true)
    } catch (error) {
        return {success: false, error: error instanceof Error ? error.message : 'Token exchange failed'}
    }

    if (!tokenResponse.access_token) {
        return {success: false, error: 'Invalid token response'}
    }

    const expiryTime = new Date().getTime() + (60 * 60 * 1000);
    localStorage.setItem('token', tokenResponse.access_token);
    localStorage.setItem('tokenExpiry', expiryTime.toString());
    return {success: true}
}