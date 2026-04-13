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
const elTplGroups     = document.getElementById("template-groups");
const elTplLoading    = document.getElementById("templates-loading");

const elPanelEmpty    = document.getElementById("panel-empty");
const elPanelForm     = document.getElementById("panel-form");
const elTemplateTitle = document.getElementById("template-title");
const elTemplateMeta  = document.getElementById("template-meta");
const elSelectRg      = document.getElementById("select-rg");
const elRgSection     = document.getElementById("rg-section");
const elParamsFields  = document.getElementById("params-fields");
const elBtnDeploy     = document.getElementById("btn-deploy");
const elDeployResult  = document.getElementById("deploy-result");
const elResultStatus  = document.getElementById("result-status");
const elResultBody    = document.getElementById("result-body");

let selectedTemplate = null;
let pollTimer = null;

const defaultIconSVG = '<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>';

/** Render the app icon into a container element. Supports image URLs, emoji, or falls back to the default SVG. */
function renderIcon(container, size) {
  const icon = window.APP_ICON;
  if (!icon) {
    container.innerHTML = defaultIconSVG;
    container.querySelector("svg").style.width = size;
    container.querySelector("svg").style.height = size;
    return;
  }
  if (icon.startsWith("http://") || icon.startsWith("https://")) {
    const img = document.createElement("img");
    img.src = icon;
    img.alt = "";
    img.style.width = size;
    img.style.height = size;
    img.style.objectFit = "contain";
    container.appendChild(img);
  } else {
    container.textContent = icon;
    container.style.fontSize = size;
    container.style.lineHeight = "1";
  }
}

// ── Bootstrap ─────────────────────────────────────────────────────────────────
window.addEventListener("DOMContentLoaded", async () => {
  renderIcon(document.getElementById("header-icon"), "20px");
  renderIcon(document.getElementById("login-icon"), "48px");

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
    alert("Could not sign in: " + e.message);
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
    const subs = (data.value || [])
      .map((s) => ({ value: s.subscriptionId, label: `${s.displayName} (${s.subscriptionId})` }))
      .sort((a, b) => a.label.localeCompare(b.label));
    populateSelect(elSelectSub, subs, "Select subscription…");
  } catch (e) {
    console.error("Failed to load subscriptions:", e);
  }
}

async function loadTemplates() {
  elTplLoading.classList.remove("hidden");
  try {
    const data = await apiGet("/api/templates", null);
    renderTemplateGroups(data.groups || []);
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
    const rgs = (data.value || [])
      .map((rg) => ({ value: rg.name, label: rg.name }))
      .sort((a, b) => a.label.localeCompare(b.label));
    populateSelect(elSelectRg, rgs, "Select resource group…");
  } catch (e) {
    console.error("Failed to load resource groups:", e);
  }
}

// ── Template groups ───────────────────────────────────────────────────────────
function renderTemplateGroups(groups) {
  elTplGroups.innerHTML = "";

  groups.forEach((group) => {
    const wrapper = document.createElement("div");
    wrapper.className = "template-group";

    const header = document.createElement("div");
    header.className = "group-header";
    header.innerHTML = `
      <svg class="group-chevron" viewBox="0 0 12 12" fill="none">
        <path d="M4 2l4 4-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      ${escapeHtml(group.name)}
    `;

    const items = document.createElement("div");
    items.className = "group-items";

    group.templates.forEach((tpl) => {
      const btn = document.createElement("button");
      btn.className = "template-item";
      btn.textContent = tpl.displayName;
      btn.title = tpl.path;
      btn.addEventListener("click", () => selectTemplate(tpl.path, btn));
      items.appendChild(btn);
    });

    header.addEventListener("click", () => {
      header.classList.toggle("collapsed");
      items.classList.toggle("hidden");
    });

    wrapper.appendChild(header);
    wrapper.appendChild(items);
    elTplGroups.appendChild(wrapper);
  });
}

async function selectTemplate(name, btnEl) {
  // Update active state
  document.querySelectorAll(".template-item.active").forEach((el) => el.classList.remove("active"));
  btnEl.classList.add("active");
  selectedTemplate = name;

  try {
    const data = await apiGet(`/api/templates/${encodeURIComponent(name)}`, null);
    renderTemplateForm(data);
  } catch (e) {
    console.error("Failed to load template params:", e);
  }
}

// ── Event handlers ────────────────────────────────────────────────────────────
async function onSubscriptionChange() {
  const subId = elSelectSub.value;
  if (!subId) return;
  await loadResourceGroups(subId);
}

function renderTemplateForm(data) {
  elTemplateTitle.textContent = data.metadata?.description || data.name;
  elParamsFields.innerHTML = "";

  // Render metadata
  renderMetadata(data.metadata || {}, data.name);

  // Auto-set deployment scope from template's targetScope
  const scope = data.targetScope || "resourceGroup";
  document.querySelectorAll('input[name="scope"]').forEach((radio) => {
    radio.checked = radio.value === scope;
  });
  elRgSection.classList.toggle("hidden", scope !== "resourceGroup");

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
    } else if (param.type === "object" || param.type === "array") {
      input = document.createElement("textarea");
      input.className = "json-input";
      input.placeholder = param.type === "object" ? '{\n  "key": "value"\n}' : '[\n  "item1",\n  "item2"\n]';
      if (param.defaultValue != null) {
        try {
          input.value = JSON.stringify(JSON.parse(param.defaultValue), null, 2);
        } catch {
          input.value = param.defaultValue;
        }
      }
      input.addEventListener("input", () => validateJsonInput(input));
    } else {
      input = document.createElement("input");
      input.className = "input";
      input.type = param.type === "int" ? "number" : "text";
      input.placeholder = param.type;
      if (param.defaultValue != null) input.value = param.defaultValue;
    }

    input.id = `param-${param.name}`;
    input.dataset.paramName = param.name;
    input.dataset.paramType = param.type;

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
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
}

function renderMetadata(meta, fileName) {
  elTemplateMeta.innerHTML = "";

  // Known display keys in preferred order
  const displayKeys = ["author", "version", "category", "tags"];
  const shown = new Set(["description"]); // description is used as title

  // Show filename always
  const fileEl = document.createElement("div");
  fileEl.className = "meta-item";
  fileEl.innerHTML = `<span class="meta-key">File</span><span class="meta-value">${escapeHtml(fileName)}</span>`;
  elTemplateMeta.appendChild(fileEl);

  // Show known keys first, then any remaining
  const allKeys = [...new Set([...displayKeys, ...Object.keys(meta)])];
  for (const key of allKeys) {
    if (shown.has(key) || !meta[key]) continue;
    shown.add(key);
    const item = document.createElement("div");
    item.className = "meta-item";
    item.innerHTML = `<span class="meta-key">${escapeHtml(key)}</span><span class="meta-value">${escapeHtml(meta[key])}</span>`;
    elTemplateMeta.appendChild(item);
  }

  elTemplateMeta.classList.toggle("hidden", Object.keys(meta).length === 0 && !fileName);
}

function validateJsonInput(textarea) {
  if (textarea.value.trim() === "") {
    textarea.classList.remove("invalid");
    return;
  }
  try {
    JSON.parse(textarea.value);
    textarea.classList.remove("invalid");
  } catch {
    textarea.classList.add("invalid");
  }
}

async function onDeploy() {
  const subId = elSelectSub.value;
  const template = selectedTemplate;
  const scope = document.querySelector('input[name="scope"]:checked').value;
  const rgName = elSelectRg.value;

  if (!subId)          return alert("Select a subscription.");
  if (!template)       return alert("Select a template.");
  if (scope === "resourceGroup" && !rgName) return alert("Select a resource group.");

  // Collect parameters with proper typing
  const parameters = {};
  let hasJsonError = false;
  document.querySelectorAll("[data-param-name]").forEach((el) => {
    const name = el.dataset.paramName;
    const type = el.dataset.paramType;
    const val = el.value;

    if (val === "") return; // skip empty — use template default

    if (type === "object" || type === "array") {
      try {
        parameters[name] = JSON.parse(val);
      } catch {
        hasJsonError = true;
        el.classList.add("invalid");
      }
    } else if (type === "int") {
      parameters[name] = parseInt(val, 10);
    } else if (type === "bool") {
      parameters[name] = val === "true";
    } else {
      parameters[name] = val;
    }
  });

  if (hasJsonError) return alert("Fix JSON errors in parameters before deploying.");

  const body = {
    templateName: template,
    scope,
    subscriptionId: subId,
    resourceGroupName: rgName,
    parameters,
  };

  elBtnDeploy.disabled = true;
  elDeployResult.classList.remove("hidden");

  // Show status timeline
  elResultStatus.className = "result-status pending";
  elResultStatus.innerHTML = `
    <div class="status-timeline">
      <div class="status-step"><span class="status-dot active"></span> Submitting deployment…</div>
    </div>`;
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
      // Get deployment status URL from backend
      const deployURL = resp.headers.get("X-Deploy-URL");
      startStatusPolling(deployURL, json);
    } else {
      showDeployError(resp.status, json);
    }
  } catch (e) {
    elResultStatus.className = "result-status error";
    elResultStatus.textContent = "✗ Network error";
    elResultBody.textContent = e.message;
  } finally {
    elBtnDeploy.disabled = false;
  }
}

// ── Deployment status polling ─────────────────────────────────────────────────

function startStatusPolling(deployURL, initialResponse) {
  if (pollTimer) clearInterval(pollTimer);

  updateDeployTimeline("Running", initialResponse);

  pollTimer = setInterval(async () => {
    try {
      const token = await getToken();
      const data = await apiGet(`/api/deploy/status?url=${encodeURIComponent(deployURL)}`, token);
      const state = data?.properties?.provisioningState || "Unknown";

      updateDeployTimeline(state, data);

      if (state === "Succeeded" || state === "Failed" || state === "Canceled") {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    } catch (e) {
      console.error("Status poll error:", e);
    }
  }, 3000);
}

function updateDeployTimeline(state, data) {
  const steps = [
    { label: "Deployment submitted", done: true },
    { label: "Validating", done: state !== "Running" || true },
    { label: "Creating resources…", active: state === "Running" },
    { label: state === "Succeeded" ? "Completed ✓" : state === "Failed" ? "Failed ✗" : "Waiting…",
      done: state === "Succeeded",
      error: state === "Failed" || state === "Canceled" },
  ];

  if (state === "Succeeded") {
    elResultStatus.className = "result-status success";
  } else if (state === "Failed" || state === "Canceled") {
    elResultStatus.className = "result-status error";
  } else {
    elResultStatus.className = "result-status pending";
  }

  elResultStatus.innerHTML = `<div class="status-timeline">${steps.map((s) => {
    let dotClass = "status-dot";
    if (s.error) dotClass += " error";
    else if (s.active) dotClass += " active";
    else if (s.done) dotClass += " done";
    return `<div class="status-step"><span class="${dotClass}"></span> ${s.label}</div>`;
  }).join("")}</div>`;

  elResultBody.textContent = JSON.stringify(data, null, 2);
}

function showDeployError(httpStatus, json) {
  elResultStatus.className = "result-status error";
  elResultStatus.textContent = `✗ Error (HTTP ${httpStatus})`;
  elResultBody.textContent = JSON.stringify(json, null, 2);
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

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}
