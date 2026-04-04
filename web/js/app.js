/**
 * app.js — Main application logic for Bicep Deployer
 */

// ── Element refs ─────────────────────────────────────────────────────────────
const elAuthOut       = document.getElementById("auth-logged-out");
const elAuthIn        = document.getElementById("auth-logged-in");
const elUserName      = document.getElementById("user-name");
const elBtnLogin      = document.getElementById("btn-login");
const elBtnLoginMain  = document.getElementById("btn-login-main");
const elBtnLogout     = document.getElementById("btn-logout");
const elSectionLogin  = document.getElementById("section-login");
const elSectionApp    = document.getElementById("section-app");

const elSelectSub     = document.getElementById("select-subscription");
const elSelectTpl     = document.getElementById("select-template");
const elTplLoading    = document.getElementById("templates-loading");

const elPanelEmpty    = document.getElementById("panel-empty");
const elPanelForm     = document.getElementById("panel-form");
const elTemplateTitle = document.getElementById("template-title");
const elSelectRg      = document.getElementById("select-rg");
const elRgSection     = document.getElementById("rg-section");
const elParamsFields  = document.getElementById("params-fields");
const elDeployName    = document.getElementById("input-deployment-name");
const elBtnDeploy     = document.getElementById("btn-deploy");
const elDeployResult  = document.getElementById("deploy-result");
const elResultStatus  = document.getElementById("result-status");
const elResultBody    = document.getElementById("result-body");

// ── Bootstrap ─────────────────────────────────────────────────────────────────
window.addEventListener("DOMContentLoaded", async () => {
  const account = await initAuth();
  if (account) {
    showApp(account);
  } else {
    showLogin();
  }

  elBtnLogin.addEventListener("click", handleLogin);
  elBtnLoginMain.addEventListener("click", handleLogin);
  elBtnLogout.addEventListener("click", handleLogout);
  elSelectSub.addEventListener("change", onSubscriptionChange);
  elSelectTpl.addEventListener("change", onTemplateChange);
  elBtnDeploy.addEventListener("click", onDeploy);

  document.querySelectorAll('input[name="scope"]').forEach((radio) => {
    radio.addEventListener("change", () => {
      elRgSection.classList.toggle("hidden", radio.value === "subscription");
    });
  });
});

// ── Auth handlers ────────────────────────────────────────────────────────────
async function handleLogin() {
  try {
    const account = await login();
    showApp(account);
  } catch (e) {
    console.error("Login error:", e);
    alert("Kunne ikke logge ind: " + e.message);
  }
}

async function handleLogout() {
  try {
    await logout();
    showLogin();
  } catch (e) {
    console.error("Logout error:", e);
  }
}

// ── UI state ─────────────────────────────────────────────────────────────────
function showLogin() {
  elSectionLogin.classList.remove("hidden");
  elSectionApp.classList.add("hidden");
  elAuthOut.classList.remove("hidden");
  elAuthIn.classList.add("hidden");
}

async function showApp(account) {
  elSectionLogin.classList.add("hidden");
  elSectionApp.classList.remove("hidden");
  elAuthOut.classList.add("hidden");
  elAuthIn.classList.remove("hidden");
  elUserName.textContent = account.name || account.username;

  await Promise.all([loadSubscriptions(), loadTemplates()]);
}

// ── Data loading ──────────────────────────────────────────────────────────────
async function loadSubscriptions() {
  try {
    const token = await getToken();
    const data = await apiGet("/api/subscriptions", token);
    populateSelect(elSelectSub, (data.value || []).map((s) => ({
      value: s.subscriptionId,
      label: `${s.displayName} (${s.subscriptionId})`,
    })), "Vælg subscription…");
  } catch (e) {
    console.error("Failed to load subscriptions:", e);
  }
}

async function loadTemplates() {
  elTplLoading.classList.remove("hidden");
  elSelectTpl.disabled = true;
  try {
    const data = await apiGet("/api/templates", null);
    populateSelect(elSelectTpl, (data.templates || []).map((t) => ({
      value: t,
      label: t,
    })), "Vælg template…");
    elSelectTpl.disabled = false;
  } catch (e) {
    console.error("Failed to load templates:", e);
  } finally {
    elTplLoading.classList.add("hidden");
  }
}

async function loadResourceGroups(subscriptionId) {
  try {
    const token = await getToken();
    const data = await apiGet(`/api/resource-groups?subscriptionId=${subscriptionId}`, token);
    populateSelect(elSelectRg, (data.value || []).map((rg) => ({
      value: rg.name,
      label: rg.name,
    })), "Vælg resource group…");
  } catch (e) {
    console.error("Failed to load resource groups:", e);
  }
}

// ── Event handlers ────────────────────────────────────────────────────────────
async function onSubscriptionChange() {
  const subId = elSelectSub.value;
  if (!subId) return;
  await loadResourceGroups(subId);
}

async function onTemplateChange() {
  const name = elSelectTpl.value;
  if (!name) {
    elPanelEmpty.classList.remove("hidden");
    elPanelForm.classList.add("hidden");
    return;
  }

  try {
    const data = await apiGet(`/api/templates/${encodeURIComponent(name)}`, null);
    renderTemplateForm(data);
  } catch (e) {
    console.error("Failed to load template params:", e);
  }
}

function renderTemplateForm(data) {
  elTemplateTitle.textContent = data.name;
  elParamsFields.innerHTML = "";

  (data.parameters || []).forEach((param) => {
    const wrapper = document.createElement("div");
    wrapper.className = "param-field";

    const label = document.createElement("label");
    label.className = "param-label";
    label.htmlFor = `param-${param.name}`;
    label.innerHTML = param.name + (param.required ? '<span class="param-required">*</span>' : "");

    let input;
    if (param.allowedValues && param.allowedValues.length > 0) {
      input = document.createElement("select");
      input.className = "select";
      param.allowedValues.forEach((v) => {
        const opt = document.createElement("option");
        opt.value = v;
        opt.textContent = v;
        if (param.defaultValue === v) opt.selected = true;
        input.appendChild(opt);
      });
    } else if (param.type === "bool") {
      input = document.createElement("select");
      input.className = "select";
      ["true", "false"].forEach((v) => {
        const opt = document.createElement("option");
        opt.value = v;
        opt.textContent = v;
        if (String(param.defaultValue) === v) opt.selected = true;
        input.appendChild(opt);
      });
    } else {
      input = document.createElement("input");
      input.className = "input";
      input.type = param.type === "int" ? "number" : "text";
      input.placeholder = param.type;
      if (param.defaultValue != null) input.value = param.defaultValue;
    }

    input.id = `param-${param.name}`;
    input.dataset.paramName = param.name;

    wrapper.appendChild(label);
    if (param.description) {
      const hint = document.createElement("p");
      hint.className = "param-hint";
      hint.textContent = param.description;
      wrapper.appendChild(hint);
    }
    wrapper.appendChild(input);
    elParamsFields.appendChild(wrapper);
  });

  elPanelEmpty.classList.add("hidden");
  elPanelForm.classList.remove("hidden");
  elDeployResult.classList.add("hidden");
}

async function onDeploy() {
  const subId = elSelectSub.value;
  const template = elSelectTpl.value;
  const deploymentName = elDeployName.value.trim();
  const scope = document.querySelector('input[name="scope"]:checked').value;
  const rgName = elSelectRg.value;

  if (!subId)          return alert("Vælg en subscription.");
  if (!template)       return alert("Vælg en template.");
  if (!deploymentName) return alert("Angiv et deployment navn.");
  if (scope === "resourceGroup" && !rgName) return alert("Vælg en resource group.");

  const parameters = {};
  document.querySelectorAll("[data-param-name]").forEach((el) => {
    parameters[el.dataset.paramName] = el.value;
  });

  const body = {
    templateName: template,
    scope,
    subscriptionId: subId,
    resourceGroupName: rgName,
    deploymentName,
    parameters,
  };

  elBtnDeploy.disabled = true;
  elDeployResult.classList.remove("hidden");
  elResultStatus.className = "result-status pending";
  elResultStatus.textContent = "Deployer…";
  elResultBody.textContent = "";

  try {
    const token = await getToken();
    const resp = await fetch("/api/deploy", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });

    const json = await resp.json();

    if (resp.ok || resp.status === 201) {
      elResultStatus.className = "result-status success";
      elResultStatus.textContent = `✓ Deployment igangsat (HTTP ${resp.status})`;
    } else {
      elResultStatus.className = "result-status error";
      elResultStatus.textContent = `✗ Fejl (HTTP ${resp.status})`;
    }
    elResultBody.textContent = JSON.stringify(json, null, 2);
  } catch (e) {
    elResultStatus.className = "result-status error";
    elResultStatus.textContent = "✗ Netværksfejl";
    elResultBody.textContent = e.message;
  } finally {
    elBtnDeploy.disabled = false;
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────
async function apiGet(path, token) {
  const headers = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  const resp = await fetch(path, { headers });
  if (!resp.ok) throw new Error(`HTTP ${resp.status} from ${path}`);
  return resp.json();
}

function populateSelect(selectEl, options, placeholder) {
  selectEl.innerHTML = `<option value="">${placeholder}</option>`;
  options.forEach(({ value, label }) => {
    const opt = document.createElement("option");
    opt.value = value;
    opt.textContent = label;
    selectEl.appendChild(opt);
  });
}
