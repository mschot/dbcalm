export const checkAuth = () => {
  const token = localStorage.getItem('token');
  const tokenExpiry = localStorage.getItem('tokenExpiry');

  if (!token || !tokenExpiry) {
    return false;
  }

  const now = new Date().getTime();
  if (now > parseInt(tokenExpiry)) {
    // Token has expired, clear it
    localStorage.removeItem('token');
    localStorage.removeItem('tokenExpiry');
    return false;
  }

  return true;
};
