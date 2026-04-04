/**
 * auth.js — MSAL.js Azure AD authentication (v3.x)
 * Tenant and Client IDs are injected by the Go server via index.html template.
 */

const msalConfig = {
  auth: {
    clientId: window.AZURE_CLIENT_ID,
    authority: `https://login.microsoftonline.com/${window.AZURE_TENANT_ID}`,
    redirectUri: window.location.origin,
  },
  cache: {
    cacheLocation: "sessionStorage",
    storeAuthStateInCookie: false,
  },
};

let msalInstance = null;

// Required scopes for Azure Management API
const ARM_SCOPES = ["https://management.azure.com/user_impersonation"];

/**
 * Initialize MSAL and handle redirect response.
 * Returns the active account if already signed in, or null.
 */
async function initAuth() {
  msalInstance = new msal.PublicClientApplication(msalConfig);
  await msalInstance.initialize();

  try {
    const response = await msalInstance.handleRedirectPromise();
    if (response) {
      msalInstance.setActiveAccount(response.account);
    }
  } catch (e) {
    console.error("MSAL redirect error:", e);
  }

  const accounts = msalInstance.getAllAccounts();
  if (accounts.length > 0) {
    msalInstance.setActiveAccount(accounts[0]);
    return accounts[0];
  }
  return null;
}

/**
 * Opens a login popup. Returns the account on success.
 */
async function login() {
  const response = await msalInstance.loginPopup({ scopes: ARM_SCOPES });
  msalInstance.setActiveAccount(response.account);
  return response.account;
}

/**
 * Logs out the current user.
 */
async function logout() {
  const account = msalInstance.getActiveAccount();
  await msalInstance.logoutPopup({ account });
}

/**
 * Returns a valid ARM access token, acquiring silently or via popup as needed.
 */
async function getToken() {
  const account = msalInstance.getActiveAccount();
  if (!account) throw new Error("Not signed in");

  try {
    const result = await msalInstance.acquireTokenSilent({
      scopes: ARM_SCOPES,
      account,
    });
    return result.accessToken;
  } catch (e) {
    const result = await msalInstance.acquireTokenPopup({ scopes: ARM_SCOPES });
    return result.accessToken;
  }
}

/**
 * Returns the currently signed-in account, or null.
 */
function getCurrentAccount() {
  return msalInstance ? msalInstance.getActiveAccount() : null;
}
