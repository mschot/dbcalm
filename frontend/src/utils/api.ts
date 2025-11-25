import { Config } from "./config"

const handleUnauthorized = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('tokenExpiry');
    window.location.href = '/login';
};

export const Api = {
    url: (path: string) => Config.apiUrl + path,

    getHeaders: (useAuth: boolean) => {
        let headers: Record<string, string> = {'Content-Type': 'application/json'}
        if (useAuth) {
            headers['Authorization'] = `Bearer ${localStorage.getItem('token')}`
        }
        return headers
    },

    get: async (path: string, useAuth: boolean = true) => {
        const response = await fetch(Api.url(path), {
            method: 'GET',
            headers: Api.getHeaders(useAuth)
        })
        if (response.ok) {
            return await response.json()
        }

        if (response.status === 401) {
            handleUnauthorized();
            throw new Error('Unauthorized');
        }

        let error = null;
        try {
            error = await response.json()
        } catch (e) {
            throw new Error(response.statusText)
        }
        throw new Error(error.detail || error.status || response.statusText)
    },

    post: async (path: string, data: object, useAuth: boolean = true, skipUnauthorizedRedirect: boolean = false) => {
        const response = await fetch(Api.url(path), {
            method: 'POST',
            headers: Api.getHeaders(useAuth),
            body: JSON.stringify(data)
        })

        if (response.ok) {
            return await response.json()
        }

        if (response.status === 401 && !skipUnauthorizedRedirect) {
            handleUnauthorized();
            throw new Error('Unauthorized');
        }

        let error = null;
        try {
            error = await response.json()
        } catch (e) {
            throw new Error(response.statusText)
        }
        throw new Error(error.detail || error.status || response.statusText)
    },

    put: async (path: string, data: object, useAuth: boolean = true) => {
        const response = await fetch(Api.url(path), {
            method: 'PUT',
            headers: Api.getHeaders(useAuth),
            body: JSON.stringify(data)
        })

        if (response.ok) {
            return await response.json()
        }

        if (response.status === 401) {
            handleUnauthorized();
            throw new Error('Unauthorized');
        }

        let error = null;
        try {
            error = await response.json()
        } catch (e) {
            throw new Error(response.statusText)
        }
        throw new Error(error.detail || error.status || response.statusText)
    },

    delete: async (path: string, useAuth: boolean = true) => {
        const response = await fetch(Api.url(path), {
            method: 'DELETE',
            headers: Api.getHeaders(useAuth)
        })

        if (response.ok) {
            // If status is 204 No Content, return an empty object
            if (response.status === 204) {
                return {}
            }
            return await response.json()
        }

        if (response.status === 401) {
            handleUnauthorized();
            throw new Error('Unauthorized');
        }

        let error = null;
        try {
            error = await response.json()
        } catch (e) {
            throw new Error(response.statusText)
        }
        throw new Error(error.detail || error.status || response.statusText)
    }
}
