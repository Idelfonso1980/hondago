const fields = ["config_path","reservar_modo","model_name","produto","group_code","cpf","cod_empre","due_day","requested_quota_id","federal_lottery","acrescimo_decrescimo","group_type","limit","cooldown_user_ms","worker_count_go","request_timeout_ms"];
    const dbFields = ["database_path","api_base_url"];
    const authFields = ["auth_id","auth_cpf","auth_cod_empresa","auth_cod_usuario","auth_cod_concessionaria","auth_senha","auth_token","auth_token_b3","auth_last_request","auth_cooldown_until","auth_blocked_until","auth_in_flight","auth_error_401_count","auth_error_429_count","auth_priority_score"];
    const solicitacaoFields = ["sol_id","sol_data_hora_solicitacao","sol_filial","sol_vendedor","sol_cpf","sol_modelo","sol_plano","sol_qtde_parcelas","sol_perc_lance","sol_com_restricao","sol_grupo","sol_observacao","sol_id_cota","sol_grupo_atendido","sol_cota_r_d","sol_data_hora_atendimento","sol_situacao","sol_lance_contemplacao"];
    const idsgFields = ["idsg_id","idsg_id_grupo","idsg_produto","idsg_vencimento","idsg_prazo","idsg_tipo","idsg_grupo","idsg_cota","idsg_r","idsg_d","idsg_parcelas_calc","idsg_booked","idsg_created_at","idsg_participantes","idsg_failed"];
    const gruposAtivosFields = ["ga_id","ga_grupo","ga_vencimento","ga_qtd_participantes","ga_data_assembleia_inaugural","ga_plano","ga_prazo","ga_tipo_grupo","ga_modelos","ga_status","ga_created_at","ga_updated_at"];
    const appUserFields = ["appuser_id","appuser_username","appuser_display_name","appuser_phone","appuser_cpf","appuser_filial","appuser_email","appuser_role","appuser_is_active","appuser_password","appuser_failed_login_attempts","appuser_locked_until","appuser_last_login_at","appuser_updated_at","appuser_created_at"];
    const solicitarFields = ["request_filial","request_data_hora_solicitacao","request_vendedor","request_cpf","request_modelo","request_plano","request_qtde_parcelas","request_perc_lance","request_com_restricao","request_grupo","request_observacao"];
    const modeloFields = ["modelo_id","modelo_idmodelo","modelo_nome","modelo_status"];
    const produtoFields = ["produto_id","produto_idproduto","produto_nome","produto_status"];
    const assembleiaFields = ["assembleia_id","assembleia_cota_r_d","assembleia_data_contemplacao","assembleia_tipo_contemplacao","assembleia_data_desclassificao","assembleia_cliente","assembleia_perc_lance","assembleia_vendedor","assembleia_grupo","assembleia_loteria_federal","assembleia_grupo_cota_r_d"];
    let monitorPollTimer = null;
    let authRows = [];
    let solicitacaoRows = [];
    let minhasSolicitacaoRows = [];
    let successMessageItems = [];
    let appUserRows = [];
    let modeloRows = [];
    let produtoRows = [];
    let assembleiaRows = [];
    let gruposAtivosRows = [];
    let idsgRows = [];
    let reservedRows = [];
    let manualNotificationRows = [];
    let authLastSearch = "";
    let appUserLastSearch = "";
    let idsgLastSearch = "";
    let reservedLastSearch = "";
    let manualNotificationLastSearch = "";
    let minhasSolicitacaoLastSearch = "";
    let modeloLastSearch = "";
    let produtoLastSearch = "";
    let assembleiaLastSearch = "";
    let gruposAtivosLastSearch = "";
    let idsgOffset = 0;
    let idsgLimit = 200;
    let idsgTotal = 0;
    let idsgLoading = false;
    let idsgHasMore = true;
    let isAuthenticated = false;
    let appBootstrapped = false;
    let currentUserRole = "";
    let currentUserName = "";
    let currentUserCPF = "";
    let currentUserFilial = "";
    let currentReserveSection = "solicitacoes";
    let dashboardLoaded = false;
    let sellerHomeLoaded = false;
    let lastStatusFullText = "Status: Pronto";
    let authRedirectInProgress = false;

    function getCookieValue(name) {
      const target = String(name || "").trim();
      if (!target) return "";
      const parts = document.cookie ? document.cookie.split(";") : [];
      for (const raw of parts) {
        const item = raw.trim();
        if (!item) continue;
        const eq = item.indexOf("=");
        if (eq <= 0) continue;
        const k = item.slice(0, eq).trim();
        if (k !== target) continue;
        return decodeURIComponent(item.slice(eq + 1));
      }
      return "";
    }

    const originalFetch = window.fetch.bind(window);
    window.fetch = function(input, init){
      const req = init ? {...init} : {};
      const method = String(req.method || "GET").toUpperCase();
      const isMutating = method === "POST" || method === "PUT" || method === "PATCH" || method === "DELETE";
      if (isMutating) {
        const csrf = getCookieValue("honda_go_csrf");
        const headers = new Headers(req.headers || {});
        if (csrf && !headers.has("X-CSRF-Token")) {
          headers.set("X-CSRF-Token", csrf);
        }
        req.headers = headers;
      }
      if (!req.credentials) {
        req.credentials = "same-origin";
      }
      return originalFetch(input, req).then((res) => {
        try {
          const url = typeof input === "string" ? input : String((input && input.url) || "");
          const isApi = url.indexOf("/api/") >= 0;
          const isAuthEndpoint =
            url.indexOf("/api/app/login") >= 0 ||
            url.indexOf("/api/app/mfa-login") >= 0 ||
            url.indexOf("/api/app/session") >= 0;
          if (isApi && !isAuthEndpoint && (res.status === 401 || res.status === 403) && isAuthenticated) {
            forceRelogin("Sessão expirada. Faça login novamente.");
          }
        } catch (_) {}
        return res;
      });
    };

    function forceRelogin(message){
      if (authRedirectInProgress) return;
      authRedirectInProgress = true;
      isAuthenticated = false;
      closeUserMenu();
      showLoginOverlay(true);
      const box = document.getElementById("loginError");
      if (box) box.textContent = message || "Sessão expirada. Faça login novamente.";
      setStatus("Deslogado");
      setTimeout(() => { authRedirectInProgress = false; }, 1200);
    }

    function formatStatusNumber(raw){
      const digits = String(raw || "").replace(/\D/g, "");
      if (!digits) return String(raw || "");
      return digits.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
    }

    function formatStatusText(text){
      const s = String(text || "");
      return s.replace(/\b\d+\b/g, (m, offset, whole) => {
        const prev = offset > 0 ? whole[offset - 1] : "";
        const next = offset + m.length < whole.length ? whole[offset + m.length] : "";
        if (prev === ":" || next === ":" || prev === "/" || next === "/" || prev === "-") {
          return m;
        }
        return formatStatusNumber(m);
      });
    }

    function applyMask(id, maskType) {
        const el = document.getElementById(id);
        if (!el) return;
        el.addEventListener("input", (e) => {
            let val = e.target.value.replace(/\D/g, "");
            if (maskType === "cpf") {
                if (val.length > 11) val = val.slice(0, 11);
                let formatted = val;
                if (val.length > 3) formatted = val.slice(0, 3) + "." + val.slice(3);
                if (val.length > 6) formatted = formatted.slice(0, 7) + "." + formatted.slice(7);
                if (val.length > 9) formatted = formatted.slice(0, 11) + "-" + formatted.slice(11);
                e.target.value = formatted;
            } else if (maskType === "phone") {
                if (val.length > 11) val = val.slice(0, 11);
                let formatted = val;
                if (val.length > 0) formatted = "(" + val;
                if (val.length > 2) formatted = "(" + val.slice(0, 2) + ") " + val.slice(2);
                if (val.length > 7) {
                    if (val.length === 11) { // 9 digits
                        formatted = "(" + val.slice(0, 2) + ") " + val.slice(2, 3) + " " + val.slice(3, 7) + "-" + val.slice(7);
                    } else { // 8 digits
                        formatted = "(" + val.slice(0, 2) + ") " + val.slice(2, 6) + "-" + val.slice(6);
                    }
                }
                e.target.value = formatted;
            }
        });
    }

    function initAllMasks() {
        applyMask("auth_cpf", "cpf");
        applyMask("appuser_cpf", "cpf");
        applyMask("request_cpf", "cpf");
        applyMask("sol_cpf", "cpf");
        applyMask("appuser_phone", "phone");
    }

    function decodeMojibake(text){
      const original = String(text || "");
      if (!original) return original;
      const suspicious = /[\u00C3\u00C2\u00C6\uFFFD]/.test(original);
      if (!suspicious) return original;

      const cp1252ToByte = {
        0x20AC: 0x80, 0x201A: 0x82, 0x0192: 0x83, 0x201E: 0x84, 0x2026: 0x85, 0x2020: 0x86, 0x2021: 0x87,
        0x02C6: 0x88, 0x2030: 0x89, 0x0160: 0x8A, 0x2039: 0x8B, 0x0152: 0x8C, 0x017D: 0x8E,
        0x2018: 0x91, 0x2019: 0x92, 0x201C: 0x93, 0x201D: 0x94, 0x2022: 0x95, 0x2013: 0x96, 0x2014: 0x97,
        0x02DC: 0x98, 0x2122: 0x99, 0x0161: 0x9A, 0x203A: 0x9B, 0x0153: 0x9C, 0x017E: 0x9E, 0x0178: 0x9F
      };

      const score = (s) => {
        let bad = 0;
        for (const ch of String(s || "")) {
          const code = ch.charCodeAt(0);
          if (code === 0x00C3 || code === 0x00C2 || code === 0x00C6 || code === 0xFFFD) bad += 2;
        }
        return bad;
      };

      const toByteArray = (s) => {
        const bytes = [];
        for (const ch of s) {
          const code = ch.charCodeAt(0);
          if (code <= 0xFF) {
            bytes.push(code);
            continue;
          }
          if (cp1252ToByte[code] !== undefined) {
            bytes.push(cp1252ToByte[code]);
            continue;
          }
          return null;
        }
        return new Uint8Array(bytes);
      };

      const decodeStep = (s) => {
        const bytes = toByteArray(s);
        if (!bytes) return null;
        return new TextDecoder("utf-8", {fatal: true}).decode(bytes);
      };

      let best = original;
      let bestScore = score(original);
      let cur = original;
      for (let i = 0; i < 4; i++) {
        let dec = null;
        try {
          dec = decodeStep(cur);
        } catch (err) {
          break;
        }
        if (!dec || dec === cur) break;
        const sc = score(dec);
        if (sc <= bestScore) {
          best = dec;
          bestScore = sc;
        }
        cur = dec;
      }
      return best;
    }
    function normalizeNodeText(root){
      if (!root) return;
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
      let node = walker.nextNode();
      while (node) {
        const cur = node.nodeValue || "";
        const dec = decodeMojibake(cur);
        if (dec !== cur) node.nodeValue = dec;
        node = walker.nextNode();
      }
    }

    function normalizeUIText(){
      const root = document.body;
      if (!root) return;
      normalizeNodeText(root);
      const attrs = ["placeholder", "title", "aria-label"];
      for (const attr of attrs) {
        const nodes = root.querySelectorAll("[" + attr + "]");
        for (const el of nodes) {
          const val = el.getAttribute(attr);
          if (val) {
            const dec = decodeMojibake(val);
            if (dec !== val) el.setAttribute(attr, dec);
          }
        }
      }
    }

    function observeUTF8Fix(){
      const root = document.body;
      if (!root || typeof MutationObserver === "undefined") return;
      let fixing = false;
      const observer = new MutationObserver((mutations) => {
        if (fixing) return;
        fixing = true;
        try {
          for (const m of mutations) {
            if (m.type === "characterData" && m.target) {
              const cur = m.target.nodeValue || "";
              const dec = decodeMojibake(cur);
              if (dec !== cur) m.target.nodeValue = dec;
            }
            for (const n of (m.addedNodes || [])) {
              if (n.nodeType === Node.TEXT_NODE) {
                const cur = n.nodeValue || "";
                const dec = decodeMojibake(cur);
                if (dec !== cur) n.nodeValue = dec;
              } else if (n.nodeType === Node.ELEMENT_NODE) {
                normalizeNodeText(n);
                const attrs = ["placeholder", "title", "aria-label"];
                for (const attr of attrs) {
                  if (n.hasAttribute && n.hasAttribute(attr)) {
                    const val = n.getAttribute(attr);
                    if (val) {
                      const dec = decodeMojibake(val);
                      if (dec !== val) n.setAttribute(attr, dec);
                    }
                  }
                  const inner = n.querySelectorAll ? n.querySelectorAll("[" + attr + "]") : [];
                  for (const el of inner) {
                    const val = el.getAttribute(attr);
                    if (val) {
                      const dec = decodeMojibake(val);
                      if (dec !== val) el.setAttribute(attr, dec);
                    }
                  }
                }
              }
            }
          }
        } finally {
          fixing = false;
        }
      });
      observer.observe(root, {childList: true, subtree: true, characterData: true});
    }

    function setStatus(text){
      const decoded = decodeMojibake(text);
      const statusBody = decoded.includes("debug_auth=") ? decoded : formatStatusText(decoded);
      const formatted = "Status: " + statusBody;
      const el = document.getElementById("status");
      if (el) el.textContent = formatted;
      lastStatusFullText = formatted;
      const detail = document.getElementById("statusDetailsText");
      if (detail && !document.getElementById("statusDetailsModal").classList.contains("hidden")) {
        detail.value = formatted;
      }
    }

    function openStatusDetailsModal(){
      const modal = document.getElementById("statusDetailsModal");
      const detail = document.getElementById("statusDetailsText");
      if (detail) detail.value = String(lastStatusFullText || "Status: Pronto");
      if (modal) modal.classList.remove("hidden");
    }

    function closeStatusDetailsModal(ev){
      if (ev && ev.target && ev.target.id !== "statusDetailsModal") return;
      const modal = document.getElementById("statusDetailsModal");
      if (modal) modal.classList.add("hidden");
    }

    async function copyStatusDetails(){
      const detail = document.getElementById("statusDetailsText");
      const text = detail ? String(detail.value || "") : "";
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
        setStatus("Status copiado para a Ã¡rea de transferÃªncia");
      } catch (err) {
        if (detail) {
          detail.focus();
          detail.select();
          try {
            document.execCommand("copy");
            setStatus("Status copiado para a Ã¡rea de transferÃªncia");
          } catch (e2) {
            setStatus("Falha ao copiar status");
          }
        }
      }
    }

    function canManageSystemUsers(){
      return String(currentUserRole || "").toLowerCase() === "admin";
    }

    function normalizeRoleValue(role){
      const v = String(role || "").trim().toLowerCase();
      if (v === "seller_name" || v === "vendedor") return "vendedor";
      if (v === "operator") return "operador";
      return v || "operador";
    }

    function isVendedorRole(){
      return normalizeRoleValue(currentUserRole) === "vendedor";
    }
    function isAdminRole(){
      return normalizeRoleValue(currentUserRole) === "admin";
    }
    function canAccessReserveHome(){
      return isVendedorRole() || isAdminRole();
    }
    function applyReserveTabOrder(){
      const container = document.querySelector(".config-sections");
      if (!container) return;
      const byName = (name) => container.querySelector("[data-reserve-tab='" + name + "']");
      let order = ["solicitacoes", "reserved", "mensagens", "config", "home", "minhas", "solicitar"];
      if (isVendedorRole()) {
        order = ["home", "minhas", "solicitar"];
      }
      for (const name of order) {
        const el = byName(name);
        if (el) container.appendChild(el);
      }
    }

    function applyRolePermissions(){
      const appUsersTab = document.querySelector("[data-config-tab='appusers']");
      const dbTab = document.querySelector("[data-config-tab='database']");
      if (appUsersTab) appUsersTab.style.display = canManageSystemUsers() ? "" : "none";
      if (dbTab) dbTab.style.display = canManageSystemUsers() ? "" : "none";
      const navDashboard = document.getElementById("nav-dashboard");
      const navReservas = document.getElementById("nav-reservas");
      const navMonitor = document.getElementById("nav-monitor");
      const navLogs = document.getElementById("nav-logs");
      const navConfig = document.getElementById("nav-config");
      if (isVendedorRole()) {
        if (navDashboard) navDashboard.style.display = "none";
        if (navMonitor) navMonitor.style.display = "none";
        if (navLogs) navLogs.style.display = "none";
        if (navConfig) navConfig.style.display = "none";
        if (navReservas) navReservas.style.display = "";
      } else {
        if (navDashboard) navDashboard.style.display = "";
        if (navReservas) navReservas.style.display = "";
        if (navMonitor) navMonitor.style.display = hasPermission("logs:read") ? "" : "none";
        if (navLogs) navLogs.style.display = hasPermission("logs:read") ? "" : "none";
        if (navConfig) navConfig.style.display = "";
      }
      applyRBAC();
      if (!canManageSystemUsers()) {
        const modal = document.getElementById("appUserEditModal");
        if (modal) modal.classList.add("hidden");
      }
    }

    function closeUserMenu(){
      const menu = document.getElementById("userMenu");
      if (menu) menu.classList.add("hidden");
    }

    function toggleUserMenu(){
      const menu = document.getElementById("userMenu");
      if (!menu) return;
      menu.classList.toggle("hidden");
    }

    function isMobileLayout(){
      return window.matchMedia("(max-width: 900px)").matches;
    }

    function updateMobileHeaderOffset(){
      const root = document.documentElement;
      const topbar = document.querySelector(".topbar");
      if (!root || !topbar) return;
      const h = Math.ceil(topbar.getBoundingClientRect().height || 76);
      // margem extra para garantir que o primeiro item do menu nao fique encoberto
      const offset = Math.max(76, h + 10);
      root.style.setProperty("--mobile-header-offset", offset + "px");
    }

    function applySidebarState(){
      updateMobileHeaderOffset();
      const collapsed = localStorage.getItem("sidebar_collapsed") === "1";
      document.body.classList.toggle("sidebar-collapsed", collapsed);
      document.body.classList.remove("sidebar-mobile-open");
    }

    function closeSidebarMobile(){
      document.body.classList.remove("sidebar-mobile-open");
    }

    function openPage(page){
      if (!isAuthenticated) {
        forceRelogin("Sessão expirada. Faça login novamente.");
        return;
      }
      if (isVendedorRole() && page !== "reservas") {
        page = "reservas";
      }
      const pages = {
        dashboard: document.getElementById("page-dashboard"),
        reservas: document.getElementById("page-reservas"),
        config: document.getElementById("page-config")
      };
      for (const [name, node] of Object.entries(pages)) {
        if (!node) continue;
        node.classList.toggle("hidden", name !== page);
      }

      const links = document.querySelectorAll("[data-page-link]");
      for (const link of links) {
        link.classList.toggle("active", link.getAttribute("data-page-link") === page);
      }
      if (page === "reservas") {
        openReserveSection(isVendedorRole() ? "home" : (currentReserveSection || "solicitacoes"));
      }
      if (page === "dashboard" && !dashboardLoaded) {
        loadDashboard();
      }
      if (page === "config" && authRows.length === 0) {
        searchAuthUsers();
      }
      if (page === "config" && appUserRows.length === 0) {
        if (canManageSystemUsers()) searchAppUsers();
      }
      if (page === "config" && idsgRows.length === 0) {
        searchIDsGrupos();
      }
      if (page === "config") {
        openConfigSection("idsgrupos");
      }
    }

    function fmtDashNumber(v){
      return formatStatusNumber(String(v || 0));
    }

    function fmtDashPercent(v){
      const n = Number(v || 0);
      if (!Number.isFinite(n)) return "0%";
      return n.toFixed(1).replace(".", ",") + "%";
    }

    function fmtDashDecimal(v, digits){
      const n = Number(v || 0);
      if (!Number.isFinite(n)) return "0";
      return n.toFixed(digits).replace(".", ",");
    }

    function fmtElapsedSeconds(raw){
      const sec = Number(raw || 0);
      if (!Number.isFinite(sec) || sec <= 0) return "--";
      if (sec < 60) return String(Math.floor(sec)) + "s";
      if (sec < 3600) return String(Math.floor(sec / 60)) + "m";
      if (sec < 86400) return String(Math.floor(sec / 3600)) + "h";
      return String(Math.floor(sec / 86400)) + "d";
    }

    function fmtDateBR(iso){
      const s = String(iso || "").trim();
      const m = s.match(/^(\d{4})-(\d{2})-(\d{2})$/);
      if (!m) return s;
      return m[3] + "/" + m[2] + "/" + m[1];
    }

    function setDashValue(id, val){
      const el = document.getElementById(id);
      if (el) el.textContent = val;
    }

    function setDashDelta(id, val){
      const el = document.getElementById(id);
      if (!el) return;
      const n = Number(val || 0);
      const sign = n > 0 ? "+" : "";
      el.textContent = sign + fmtDashDecimal(n, 1) + "% vs perÃ­odo anterior";
      el.classList.remove("up", "down");
      if (n > 0.05) el.classList.add("up");
      else if (n < -0.05) el.classList.add("down");
    }

    function renderDashRankTable(tbodyId, rows, showAtendida, detailKind){
      const tbody = document.getElementById(tbodyId);
      if (!tbody) return;
      if (!Array.isArray(rows) || !rows.length) {
        tbody.innerHTML = "<tr><td colspan=\"" + (showAtendida ? "4" : "2") + "\">Sem dados</td></tr>";
        return;
      }
      tbody.innerHTML = rows.map((r) => {
        const name = escapeHtml(r.nome || "-");
        const encVal = encodeURIComponent(String(r.nome || ""));
        const clickable = detailKind ? ("<button type=\"button\" class=\"dash-link-btn\" onclick=\"openDashboardDetails('" + detailKind + "', decodeURIComponent('" + encVal + "'))\">" + name + "</button>") : name;
        if (showAtendida) {
          return "<tr>" +
            "<td>" + clickable + "</td>" +
            "<td>" + fmtDashNumber(r.total || 0) + "</td>" +
            "<td>" + fmtDashNumber(r.atendidas || 0) + "</td>" +
            "<td>" + fmtDashPercent(r.taxa || 0) + "</td>" +
          "</tr>";
        }
        return "<tr>" +
          "<td>" + clickable + "</td>" +
          "<td>" + fmtDashNumber(r.total || 0) + "</td>" +
        "</tr>";
      }).join("");
    }

    function renderDashSeries(items, labelKey){
      const root = document.getElementById("dash_series");
      if (!root) return;
      if (!Array.isArray(items) || !items.length) {
        root.innerHTML = "<div class=\"dash-bar-row\"><div>Sem dados</div><div></div><div></div></div>";
        return;
      }
      let maxVal = 0;
      for (const it of items) {
        const n = Number(it.solicitadas || 0);
        if (n > maxVal) maxVal = n;
      }
      if (maxVal <= 0) maxVal = 1;
      root.innerHTML = items.map((it) => {
        const sol = Number(it.solicitadas || 0);
        const ate = Number(it.atendidas || 0);
        const pct = Math.max(2, Math.round((sol / maxVal) * 100));
        let label = (labelKey && it[labelKey]) ? String(it[labelKey]) : String(it.holiday_date || "-");
        if (labelKey === "holiday_date") {
          label = fmtDateBR(label);
        }
        return "<div class=\"dash-bar-row\">" +
          "<div>" + escapeHtml(label) + "</div>" +
          "<div class=\"dash-bar-track\"><div class=\"dash-bar-fill\" style=\"width:" + String(pct) + "%\"></div></div>" +
          "<div>" + fmtDashNumber(ate) + "/" + fmtDashNumber(sol) + "</div>" +
        "</div>";
      }).join("");
    }

    function renderDashSeriesTo(rootId, items, labelKey){
      const prev = document.getElementById("dash_series");
      const target = document.getElementById(rootId);
      if (!target) return;
      if (prev && prev.id !== rootId) prev.id = "__dash_series_tmp";
      target.id = "dash_series";
      renderDashSeries(items, labelKey);
      target.id = rootId;
      if (prev && prev.id === "__dash_series_tmp") prev.id = "dash_series";
    }

    function fillDashFilialOptions(list){
      const sel = document.getElementById("dash_filial");
      if (!sel) return;
      const current = sel.value || "";
      const items = Array.isArray(list) ? list : [];
      let html = "<option value=\"\">Todas</option>";
      for (const f of items) {
        const v = String(f || "").trim();
        if (!v) continue;
        html += "<option value=\"" + escapeHtml(v) + "\">" + escapeHtml(v) + "</option>";
      }
      sel.innerHTML = html;
      sel.value = current;
    }

    function closeDashboardDetailsModal(ev){
      if (ev && ev.target && ev.target.id !== "dashboardDetailsModal") return;
      const modal = document.getElementById("dashboardDetailsModal");
      if (modal) modal.classList.add("hidden");
    }

    async function openDashboardDetails(kind, value){
      const titleMap = {
        total: "Detalhes: SolicitaÃ§Ãµes",
        atendidas: "Detalhes: Atendidas",
        nao_atendidas: "Detalhes: NÃ£o Atendidas",
        Filial: "Detalhes por Filial",
        seller_name: "Detalhes por Vendedor",
        group_code: "Detalhes por Grupo",
        filial_backlog: "Detalhes Backlog por Filial"
      };
      const title = document.getElementById("dashDetailsTitle");
      if (title) title.textContent = titleMap[kind] || "Detalhamento";
      const body = document.getElementById("dashDetailsBody");
      if (body) body.innerHTML = "<tr><td colspan=\"5\">Carregando...</td></tr>";
      const modal = document.getElementById("dashboardDetailsModal");
      if (modal) modal.classList.remove("hidden");

      const params = new URLSearchParams(dashboardQueryString());
      params.set("kind", kind || "total");
      if (value) params.set("value", String(value));
      const res = await fetch("/api/dashboard/details?" + params.toString());
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        if (body) body.innerHTML = "<tr><td colspan=\"5\">" + escapeHtml(data.message || "Erro ao carregar detalhes") + "</td></tr>";
        return;
      }
      const items = Array.isArray(data.items) ? data.items : [];
      if (!items.length) {
        if (body) body.innerHTML = "<tr><td colspan=\"5\">Sem dados.</td></tr>";
        return;
      }
      body.innerHTML = items.map((s) => {
        return "<tr>" +
          "<td>" + escapeHtml(String(s.id || "")) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(s.requested_at || "")) + "</td>" +
          "<td>" + escapeHtml(s.branch || "") + "</td>" +
          "<td>" + escapeHtml(s.seller_name || "") + "</td>" +
          "<td>" + escapeHtml(String(s.served_group || "")) + "</td>" +
        "</tr>";
      }).join("");
    }

    function computeApiHealthFromLogs(rawLogs, fromDate, toDate){
      const lines = String(rawLogs || "").split(/\r?\n/).filter((x) => x.trim().length > 0);
      const dateFrom = String(fromDate || "").trim();
      const dateTo = String(toDate || "").trim();
      let sumLatency = 0;
      let countLatency = 0;
      let timeoutCount = 0;
      let count4xx = 0;
      let count5xx = 0;
      for (const line of lines) {
        const mDate = line.match(/^(\d{4})\/(\d{2})\/(\d{2})/);
        if (mDate) {
          const d = mDate[1] + "-" + mDate[2] + "-" + mDate[3];
          if (dateFrom && d < dateFrom) continue;
          if (dateTo && d > dateTo) continue;
        }
        const lower = line.toLowerCase();
        if (lower.includes("timeout")) timeoutCount++;
        const mCode = line.match(/\bstatus=(\d{3})\b/i) || line.match(/\bresponse\s+(\d{3})\b/i);
        if (mCode) {
          const code = Number(mCode[1]);
          if (code >= 400 && code < 500) count4xx++;
          if (code >= 500 && code < 600) count5xx++;
        }
        const mMs = line.match(/\bfinished in ([0-9]+(?:\.[0-9]+)?)ms\b/i);
        if (mMs) {
          sumLatency += Number(mMs[1] || 0);
          countLatency++;
        }
      }
      const avg = countLatency > 0 ? (sumLatency / countLatency) : 0;
      return { avgMs: avg, timeoutCount, count4xx, count5xx };
    }

    function toLocalDateInputValue(d){
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, "0");
      const day = String(d.getDate()).padStart(2, "0");
      return year + "-" + month + "-" + day;
    }

    function setDashboardDefaultDates(){
      const fromEl = document.getElementById("dash_from");
      const toEl = document.getElementById("dash_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }

    function setSolicitacoesDefaultDates(){
      const fromEl = document.getElementById("sol_from");
      const toEl = document.getElementById("sol_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }

    function setReservedDefaultDates(){
      const fromEl = document.getElementById("reserved_from");
      const toEl = document.getElementById("reserved_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }

    function setMinhasSolicitacoesDefaultDates(){
      const fromEl = document.getElementById("my_sol_from");
      const toEl = document.getElementById("my_sol_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }

    function setMensagensDefaultDates(){
      const fromEl = document.getElementById("msg_from");
      const toEl = document.getElementById("msg_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }
    function setSellerHomeDefaultDates(){
      const fromEl = document.getElementById("home_from");
      const toEl = document.getElementById("home_to");
      if (!fromEl || !toEl) return;
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      if (!fromEl.value) fromEl.value = toLocalDateInputValue(monthStart);
      if (!toEl.value) toEl.value = toLocalDateInputValue(now);
    }

    function sellerHomeGreeting(){
      const h = new Date().getHours();
      if (h < 12) return "Bom dia";
      if (h < 18) return "Boa tarde";
      return "Boa noite";
    }
    function applySellerHomeLinkRules(){
      const sugestao = document.getElementById("seller_link_sugestao");
      const propostas = document.getElementById("seller_link_propostas");
      if (!sugestao || !propostas) return;
      const blocked = isVendedorRole() && String(currentUserFilial || "").trim().toUpperCase() !== "TER";
      for (const el of [sugestao, propostas]) {
        el.classList.toggle("disabled", blocked);
        if (blocked) {
          el.setAttribute("aria-disabled", "true");
          el.setAttribute("title", "Disponível apenas para vendedor da filial TER");
        } else {
          el.removeAttribute("aria-disabled");
          el.removeAttribute("title");
        }
      }
    }
    function applySellerHomeAdminFilters(){
      const wrap = document.getElementById("home_vendor_field");
      const input = document.getElementById("home_vendor");
      const allow = isAdminRole();
      if (wrap) wrap.classList.toggle("hidden", !allow);
      if (input && !allow) input.value = "";
    }

    function parseSellerStatusNorm(raw){
      const s = String(raw || "").trim().toLowerCase();
      if (!s) return "";
      if (s.includes("atend")) return "atendida";
      if (s.includes("solicit")) return "solicitada";
      if (s.includes("digit")) return "digitada";
      if (s.includes("expir")) return "expirada";
      return s;
    }

    async function loadSellerHome(){
      setSellerHomeDefaultDates();
      const from = (document.getElementById("home_from")?.value || "").trim();
      const to = (document.getElementById("home_to")?.value || "").trim();
      const sellerQ = isAdminRole() ? (document.getElementById("home_vendor")?.value || "").trim() : "";
      const url = "/api/solicitacoes/minhas?status=all&q=" + encodeURIComponent(sellerQ) + "&from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to);
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao carregar home do vendedor");
        return;
      }
      const items = Array.isArray(data.items) ? data.items : [];
      let atendidas = 0;
      let slaSumMin = 0;
      let slaCount = 0;
      for (const it of items) {
        const statusNorm = parseSellerStatusNorm(it.status);
        const isAtendida = statusNorm === "atendida";
        if (isAtendida) atendidas++;
        const start = parseDateTimeFlexible(it.requested_at || "");
        const end = parseDateTimeFlexible(it.served_at || "");
        if (isAtendida && start && end) {
          const diff = Math.floor((end.getTime() - start.getTime()) / 60000);
          if (Number.isFinite(diff) && diff >= 0) {
            slaSumMin += diff;
            slaCount++;
          }
        }
      }
      const total = items.length;
      const taxa = total > 0 ? (atendidas * 100) / total : 0;
      const slaMedio = slaCount > 0 ? (slaSumMin / slaCount) : 0;
      const set = (id, value) => {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
      };
      set("seller_home_total", formatStatusNumber(String(total)));
      set("seller_home_atendidas", formatStatusNumber(String(atendidas)));
      set("seller_home_taxa", (Number.isFinite(taxa) ? taxa : 0).toFixed(1).replace(".", ",") + "%");
      set("seller_home_sla", (Number.isFinite(slaMedio) ? slaMedio : 0).toFixed(1).replace(".", ",") + " min");
      sellerHomeLoaded = true;
      setStatus("Home atualizada");
    }

    function dashboardQueryString(){
      const params = new URLSearchParams();
      const from = (document.getElementById("dash_from")?.value || "").trim();
      const to = (document.getElementById("dash_to")?.value || "").trim();
      const branch = (document.getElementById("dash_filial")?.value || "").trim();
      if (branch) params.set("branch", branch);
      if (from) params.set("from", from);
      if (to) params.set("to", to);
      return params.toString();
    }

    async function loadDashboard(){
      setStatus("Carregando dashboard");
      const qs = dashboardQueryString();
      const [res, logsRes] = await Promise.all([
        fetch("/api/dashboard/summary?" + qs),
        fetch("/api/logs"),
      ]);
      const data = await res.json();
      const logsData = await logsRes.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao carregar dashboard");
        return;
      }
      const c = data.cards || {};
      const cmp = data.comparativo || {};
      setDashValue("dash_total", fmtDashNumber(c.total_solicitacoes || 0));
      setDashValue("dash_atendidas", fmtDashNumber(c.atendidas || 0));
      setDashValue("dash_nao_atendidas", fmtDashNumber(c.nao_atendidas || 0));
      setDashValue("dash_taxa", fmtDashPercent(c.taxa_atendimento || 0));
      setDashValue("dash_sla", fmtDashDecimal(c.sla_medio_min || 0, 1));
      setDashValue("dash_lance", fmtDashPercent(c.lance_medio || 0));
      setDashValue("dash_ultima_delta", fmtElapsedSeconds(c.tempo_desde_ultima_atendida_seg || 0));
      setDashValue("dash_ultima_em", c.ultima_atendida_em ? ("em " + formatDateTimeBRSeconds(c.ultima_atendida_em)) : "");
      setDashDelta("dash_total_delta", (cmp.total_solicitacoes || {}).delta_pct || 0);
      setDashDelta("dash_atendidas_delta", (cmp.atendidas || {}).delta_pct || 0);
      setDashDelta("dash_nao_atendidas_delta", (cmp.nao_atendidas || {}).delta_pct || 0);
      setDashDelta("dash_taxa_delta", (cmp.taxa_atendimento || {}).delta_pct || 0);
      setDashDelta("dash_sla_delta", (cmp.sla_medio_min || {}).delta_pct || 0);
      setDashDelta("dash_lance_delta", (cmp.lance_medio || {}).delta_pct || 0);
      renderDashRankTable("dash_filiais", data.filiais || [], true, "branch");
      renderDashRankTable("dash_vendedores", data.vendedores || [], true, "seller_name");
      renderDashRankTable("dash_vendedores_taxa", data.vendedores_taxa || [], true, "seller_name");
      renderDashRankTable("dash_grupos", data.grupos_atendidos || [], false, "group_code");
      renderDashRankTable("dash_filiais_backlog", data.filiais_backlog || [], false, "filial_backlog");
      renderDashSeriesTo("dash_series", data.serie || [], "holiday_date");
      renderDashSeriesTo("dash_series_hour", data.serie_hora || [], "hora");
      fillDashFilialOptions(data.filiais_filtro || []);
      const p = data.periodo || {};
      const health = computeApiHealthFromLogs(logsData.logs || "", p.from || "", p.to || "");
      setDashValue("dash_api_latency", fmtDashDecimal(health.avgMs || 0, 1) + " ms");
      setDashValue("dash_api_timeout", fmtDashNumber(health.timeoutCount || 0));
      setDashValue("dash_api_4xx", fmtDashNumber(health.count4xx || 0));
      setDashValue("dash_api_5xx", fmtDashNumber(health.count5xx || 0));
      const taxaCard = document.getElementById("dash_card_taxa");
      if (taxaCard) taxaCard.classList.toggle("dash-alert", Number(c.taxa_atendimento || 0) < 80);
      dashboardLoaded = true;
      const from = String(p.from || "");
      const to = String(p.to || "");
      setStatus("Dashboard atualizado (" + fmtDateBR(from) + " a " + fmtDateBR(to) + ")");
    }

    function openReserveSection(section){
      applyReserveTabOrder();
      const home = document.getElementById("reserve-home");
      const solicitacoes = document.getElementById("reserve-solicitacoes");
      const minhas = document.getElementById("reserve-minhas");
      const reserved = document.getElementById("reserve-reserved");
      const mensagens = document.getElementById("reserve-mensagens");
      const config = document.getElementById("reserve-config");
      const solicitar = document.getElementById("reserve-solicitar");
      const logPanel = document.getElementById("reserve-log-panel");
      const page = document.getElementById("page-reservas");

      if (isVendedorRole() && section !== "solicitar" && section !== "minhas" && section !== "home") {
        section = "home";
      }
      if (!canAccessReserveHome() && section === "home") {
        section = "solicitacoes";
      }

      if (home) home.classList.toggle("hidden", section !== "home");
      if (solicitacoes) solicitacoes.classList.toggle("hidden", section !== "solicitacoes");
      if (minhas) minhas.classList.toggle("hidden", section !== "minhas");
      if (reserved) reserved.classList.toggle("hidden", section !== "reserved");
      if (mensagens) mensagens.classList.toggle("hidden", section !== "mensagens");
      if (config) config.classList.toggle("hidden", section !== "config");
      if (solicitar) solicitar.classList.toggle("hidden", section !== "solicitar");
      if (logPanel) logPanel.classList.toggle("hidden", section !== "config");
      if (page) page.classList.toggle("reserve-wide", section !== "config");

      const tabs = document.querySelectorAll("[data-reserve-tab]");
      for (const tab of tabs) {
        const tabName = tab.getAttribute("data-reserve-tab");
        if (isVendedorRole() && tabName !== "solicitar" && tabName !== "minhas" && tabName !== "home") {
          tab.style.display = "none";
        } else {
          tab.style.display = "";
        }
        tab.classList.toggle("active", tabName === section);
      }
      if (canAccessReserveHome()) {
        const homeTab = document.querySelector("[data-reserve-tab='home']");
        const minhasTab = document.querySelector("[data-reserve-tab='minhas']");
        const solicitarTab = document.querySelector("[data-reserve-tab='solicitar']");
        if (homeTab && minhasTab && homeTab.parentNode === minhasTab.parentNode) {
          homeTab.classList.remove("hidden");
          homeTab.parentNode.insertBefore(homeTab, minhasTab);
        }
        if (minhasTab && solicitarTab && minhasTab.parentNode === solicitarTab.parentNode) {
          minhasTab.parentNode.insertBefore(minhasTab, solicitarTab);
        }
        applySellerHomeAdminFilters();
      }
      if (!canAccessReserveHome()) {
        const homeTab = document.querySelector("[data-reserve-tab='home']");
        if (homeTab) homeTab.classList.add("hidden");
        const minhasTab = document.querySelector("[data-reserve-tab='minhas']");
        const solicitarTab = document.querySelector("[data-reserve-tab='solicitar']");
        if (minhasTab && solicitarTab && minhasTab.parentNode === solicitarTab.parentNode) {
          minhasTab.parentNode.insertBefore(minhasTab, solicitarTab);
        }
      }

      currentReserveSection = section;

      if (section === "home") {
        setSellerHomeDefaultDates();
        applySellerHomeLinkRules();
        loadSellerHome();
      }
      if (section === "solicitacoes" && solicitacaoRows.length === 0) {
        searchSolicitacoes();
      }
      if (section === "minhas") {
        searchMinhasSolicitacoes();
      }
      if (section === "reserved" && reservedRows.length === 0) {
        searchReservedCotas();
      }
      if (section === "mensagens" && manualNotificationRows.length === 0) {
        setMensagensDefaultDates();
        searchManualNotifications();
      }
      if (section === "solicitar") {
        const isVendedor = isVendedorRole();
        const branch = document.getElementById("request_filial");
        const dt = document.getElementById("request_data_hora_solicitacao");
        const vendor = document.getElementById("request_vendedor");
        const cpf = document.getElementById("request_cpf");

        if (branch) branch.readOnly = isVendedor;
        if (dt) dt.readOnly = isVendedor;
        if (vendor) vendor.readOnly = isVendedor;
        if (cpf) cpf.readOnly = isVendedor;

        [branch, dt, vendor, cpf].forEach(el => {
          if (el) el.classList.toggle("readonly", isVendedor);
        });

        clearSolicitarCotaForm();
        loadSolicitarModeloOptions();
      }
    }

    function openConfigSection(section){
      if ((section === "appusers" || section === "database" || section === "rbac") && !canManageSystemUsers()) {
        setStatus("Acesso negado: perfil sem permissÃ£o");
        section = "users";
      }
      const appusers = document.getElementById("config-appusers");
      const rbac = document.getElementById("config-rbac");
      const users = document.getElementById("config-users");
      const idsgrupos = document.getElementById("config-idsgrupos");
      const active_groups = document.getElementById("config-active_groups");
      const assemblies = document.getElementById("config-assemblies");
      const models = document.getElementById("config-models");
      const produtos = document.getElementById("config-produtos");
      const audit = document.getElementById("config-audit");
      const database = document.getElementById("config-database");
      if (appusers) appusers.classList.toggle("hidden", section !== "appusers");
      if (rbac) {
        rbac.classList.toggle("hidden", section !== "rbac");
        if (section === "rbac") loadRBACMatrix();
      }
      if (users) users.classList.toggle("hidden", section !== "users");
      if (idsgrupos) idsgrupos.classList.toggle("hidden", section !== "idsgrupos");
      if (active_groups) active_groups.classList.toggle("hidden", section !== "active_groups");
      if (assemblies) assemblies.classList.toggle("hidden", section !== "assemblies");
      if (models) models.classList.toggle("hidden", section !== "models");
      if (produtos) produtos.classList.toggle("hidden", section !== "produtos");
      if (audit) {
        audit.classList.toggle("hidden", section !== "audit");
        if (section === "audit") loadAuditLogs();
      }
      if (database) database.classList.toggle("hidden", section !== "database");

      const tabs = document.querySelectorAll("[data-config-tab]");
      for (const tab of tabs) {
        tab.classList.toggle("active", tab.getAttribute("data-config-tab") === section);
      }
      if (section === "appusers" && appUserRows.length === 0) {
        searchAppUsers();
      }
      if (section === "models" && modeloRows.length === 0) {
        searchModelos();
      }
      if (section === "produtos" && produtoRows.length === 0) {
        searchProdutos();
      }
      if (section === "assemblies" && assembleiaRows.length === 0) {
        searchAssembleias();
      }
      if (section === "active_groups" && gruposAtivosRows.length === 0) {
        searchGruposAtivos();
      }
    }

    function toggleSidebar(){
      updateMobileHeaderOffset();
      if (isMobileLayout()) {
        const open = !document.body.classList.contains("sidebar-mobile-open");
        document.body.classList.toggle("sidebar-mobile-open", open);
        return;
      }
      const collapsed = !document.body.classList.contains("sidebar-collapsed");
      document.body.classList.toggle("sidebar-collapsed", collapsed);
      localStorage.setItem("sidebar_collapsed", collapsed ? "1" : "0");
    }

    function gather(){
      const out = {};
      for (const field of fields) out[field] = document.getElementById(field).value;
      for (const field of dbFields) {
        const el = document.getElementById(field);
        if (el) out[field] = el.value;
      }
      out.dry_run = document.getElementById("dry_run").checked;
      return out;
    }

    function fill(data){
      for (const field of fields) {
        if (data[field] !== undefined) document.getElementById(field).value = data[field];
      }
      for (const field of dbFields) {
        if (data[field] !== undefined) {
          const el = document.getElementById(field);
          if (el) el.value = data[field];
        }
      }
      document.getElementById("dry_run").checked = !!data.dry_run;
    }

    function clearAuthForm(){
      for (const field of authFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
    }

    function fillAuthForm(data){
      if (!data) {
        clearAuthForm();
        return;
      }
      document.getElementById("auth_id").value = data.id ?? "";
      document.getElementById("auth_cpf").value = data.cpf ?? "";
      document.getElementById("auth_cod_empresa").value = data.company_code ?? "";
      document.getElementById("auth_cod_usuario").value = data.account_user ?? "";
      document.getElementById("auth_cod_concessionaria").value = data.dealer_code ?? "";
      document.getElementById("auth_senha").value = "";
      document.getElementById("auth_token").value = data.token ?? "";
      document.getElementById("auth_token_b3").value = data.b3_token ?? "";
      document.getElementById("auth_last_request").value = data.last_request_at ?? "";
      document.getElementById("auth_cooldown_until").value = data.cooldown_until ?? "";
      document.getElementById("auth_blocked_until").value = data.blocked_until ?? "";
      document.getElementById("auth_in_flight").value = data.in_flight ?? "";
      document.getElementById("auth_error_401_count").value = data.error_401_count ?? "";
      document.getElementById("auth_error_429_count").value = data.error_429_count ?? "";
      document.getElementById("auth_priority_score").value = data.priority_score ?? "";
    }

    function gatherAuthForm(){
      return {
        id: Number(document.getElementById("auth_id").value || 0),
        cpf: (document.getElementById("auth_cpf").value || "").replace(/\D/g, ""),
        company_code: document.getElementById("auth_cod_empresa").value || "",
        account_user: document.getElementById("auth_cod_usuario").value || "",
        dealer_code: document.getElementById("auth_cod_concessionaria").value || "",
        account_password: document.getElementById("auth_senha").value || ""
      };
    }

    function clearAppUserForm(){
      for (const field of appUserFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const active = document.getElementById("appuser_is_active");
      if (active) active.value = "1";
      const role = document.getElementById("appuser_role");
      if (role) role.value = "operador";
    }

    function fillAppUserForm(data){
      if (!data) {
        clearAppUserForm();
        return;
      }
      document.getElementById("appuser_id").value = data.id || "";
      document.getElementById("appuser_username").value = data.username || "";
      document.getElementById("appuser_display_name").value = data.full_name || "";
      document.getElementById("appuser_cpf").value = data.cpf || "";
      document.getElementById("appuser_filial").value = data.branch || data.filial || "";
      document.getElementById("appuser_phone").value = data.phone || "";
      document.getElementById("appuser_email").value = data.email || "";
      document.getElementById("appuser_role").value = normalizeRoleValue(data.role || "operador");
      document.getElementById("appuser_is_active").value = data.is_active ? "1" : "0";
      document.getElementById("appuser_password").value = "";
      document.getElementById("appuser_failed_login_attempts").value = data.failed_login_attempts || 0;
      document.getElementById("appuser_locked_until").value = data.locked_until || "";
      document.getElementById("appuser_last_login_at").value = data.last_login_at || "";
      document.getElementById("appuser_updated_at").value = data.updated_at || "";
      document.getElementById("appuser_created_at").value = data.created_at || "";
    }

    function gatherAppUserForm(){
      return {
        id: Number(document.getElementById("appuser_id").value || 0),
        username: document.getElementById("appuser_username").value || "",
        full_name: document.getElementById("appuser_display_name").value || "",
        cpf: (document.getElementById("appuser_cpf").value || "").replace(/\D/g, ""),
        branch: document.getElementById("appuser_filial").value || "",
        email: document.getElementById("appuser_email").value || "",
        phone: (document.getElementById("appuser_phone").value || "").replace(/\D/g, ""),
        role: normalizeRoleValue(document.getElementById("appuser_role").value || "operador"),
        is_active: document.getElementById("appuser_is_active").value === "1",
        password: document.getElementById("appuser_password").value || ""
      };
    }

    function openAppUserCreateModal(){
      clearAppUserForm();
      const t = document.getElementById("appUserModalTitle");
      if (t) t.textContent = "Novo UsuÃ¡rio do Sistema";
      document.getElementById("appUserEditModal").classList.remove("hidden");
    }

    function closeAppUserEditModal(ev){
      if (ev && ev.target && ev.target.id !== "appUserEditModal") return;
      document.getElementById("appUserEditModal").classList.add("hidden");
    }

    async function openAppUserEditModal(id){
      if (!canManageSystemUsers()) {
        setStatus("Acesso negado: perfil sem permissÃ£o");
        return;
      }
      const t = document.getElementById("appUserModalTitle");
      if (t) t.textContent = "Editar UsuÃ¡rio do Sistema";
      const res = await fetch("/api/appuser/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar usuÃ¡rio");
        return;
      }
      fillAppUserForm(data);
      document.getElementById("appUserEditModal").classList.remove("hidden");
      setStatus("UsuÃ¡rio do sistema carregado");
    }

    function renderAppUserTable(items){
      appUserRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("appuserTableBody");
      if (!appUserRows.length) {
        body.innerHTML = "<tr><td colspan=\"11\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      const rows = appUserRows.map((u) => {
        const id = Number(u.id || 0);
        const active = u.is_active ? "Sim" : "NÃ£o";
        return "<tr>" +
          "<td>" + escapeHtml(id) + "</td>" +
          "<td>" + escapeHtml(u.username || "") + "</td>" +
          "<td>" + escapeHtml(u.full_name || "") + "</td>" +
          "<td>" + escapeHtml(u.cpf || "") + "</td>" +
          "<td>" + escapeHtml(u.branch || u.filial || "") + "</td>" +
          "<td>" + escapeHtml(u.email || "") + "</td>" +
          "<td>" + escapeHtml(u.phone || "") + "</td>" +
          "<td>" + escapeHtml(normalizeRoleValue(u.role || "")) + "</td>" +
          "<td>" + escapeHtml(active) + "</td>" +
          "<td>" + escapeHtml(u.last_login_at || "") + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openAppUserEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteAppUser(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
      body.innerHTML = rows;
    }

    async function searchAppUsers(){
      if (!canManageSystemUsers()) {
        renderAppUserTable([]);
        return;
      }
      const q = (document.getElementById("appuser_search").value || "").trim();
      appUserLastSearch = q;
      setStatus("Buscando usuÃ¡rios do sistema");
      const res = await fetch("/api/appusers?q=" + encodeURIComponent(q));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro na busca de usuÃ¡rios do sistema");
        renderAppUserTable([]);
        return;
      }
      renderAppUserTable(data.items || []);
      setStatus("UsuÃ¡rios do sistema carregados: " + String(data.count || 0));
    }

    async function saveAppUser(){
      if (!canManageSystemUsers()) {
        setStatus("Acesso negado: perfil sem permissÃ£o");
        return;
      }
      const payload = gatherAppUserForm();
      if (!payload.username || !payload.role) {
        setStatus("Preencha nome de usuÃ¡rio e perfil");
        return;
      }
      if (!payload.id && !payload.password) {
        setStatus("Senha obrigatÃ³ria para novo usuÃ¡rio");
        return;
      }
      setStatus("Salvando usuÃ¡rio do sistema");
      const res = await fetch("/api/appuser/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar usuÃ¡rio do sistema");
        return;
      }
      setStatus(data.message || "UsuÃ¡rio do sistema salvo");
      await searchAppUsers();
      if (data.id) await openAppUserEditModal(data.id);
    }

    async function deleteAppUser(id){
      if (!canManageSystemUsers()) {
        setStatus("Acesso negado: perfil sem permissÃ£o");
        return;
      }
      if (!id) return;
      if (!confirm("Confirma excluir este usuario do sistema")) return;
      const res = await fetch("/api/appuser/delete?id=" + encodeURIComponent(String(id)), {method: "POST"});
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir usuÃ¡rio do sistema");
        return;
      }
      setStatus(data.message || "UsuÃ¡rio do sistema removido");
      await searchAppUsers();
    }

    function showLoginOverlay(show){
      const ov = document.getElementById("loginOverlay");
      if (!ov) return;
      ov.classList.toggle("hidden", !show);
    }

    async function ensureAppSession(){
      const res = await fetch("/api/app/session");
      if (!res.ok) return false;
      const data = await res.json();
      if (!data || data.ok === false) {
        isAuthenticated = false;
        currentUserRole = "";
        sellerHomeLoaded = false;
        window.currentUserRole = "";
        window.userPermissions = [];
        return false;
      }
      isAuthenticated = true;
      currentUserRole = normalizeRoleValue(data.role || "");
      window.currentUserRole = currentUserRole;
      window.userPermissions = data.permissions || [];
      currentUserName = String(data.full_name || data.username || "");
      currentUserCPF = String(data.cpf || "");
      currentUserFilial = String(data.branch || data.filial || "");
      sellerHomeLoaded = false;
      const hello = document.querySelector(".user-chip span:last-child");
      if (hello) hello.textContent = "OlÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡, " + String(data.full_name || data.username || "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio");
      applyRolePermissions();
      return true;
    }

    async function loginAppUser(){
      const username = (document.getElementById("login_username").value || "").trim();
      const password = document.getElementById("login_password").value || "";
      const loginErr = document.getElementById("login_error");
      if (loginErr) loginErr.textContent = "";
      if (!username || !password) {
        const msg = "Informe usuÃ¡rio e senha";
        if (loginErr) loginErr.textContent = msg;
        setStatus(msg);
        return;
      }
      const res = await fetch("/api/app/login?_cb=" + Date.now(), {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({username, password})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        if (res.status === 401) {
          const msg = "UsuÃ¡rio ou senha incorretos. Revise os dados antes de tentar novamente.";
          if (loginErr) loginErr.textContent = msg;
          setStatus(msg);
        } else {
          let msg = data.message || "Falha no login";
          if (res.status >= 500) {
            msg = "ServiÃ§o temporariamente indisponÃ­vel. Tente novamente em instantes.";
          }
          if (loginErr) loginErr.textContent = msg;
          setStatus(msg);
        }
        return;
      }

      if (data.mfa_required) {
        mfaTempToken = data.temp_token;
        document.getElementById("loginMainFields").classList.add("hidden");
        document.getElementById("mfaChallengeFields").classList.remove("hidden");
        document.getElementById("login_mfa_code").value = "";
        document.getElementById("login_mfa_code").focus();
        setStatus("Segundo fator de autenticaÃ§Ã£o necessÃ¡rio");
        return;
      }

      isAuthenticated = true;
      currentUserRole = normalizeRoleValue(data.role || "");
      window.currentUserRole = currentUserRole;
      window.userPermissions = data.permissions || [];
      currentUserName = String(data.full_name || data.username || "");
      currentUserCPF = String(data.cpf || "");
      currentUserFilial = String(data.branch || data.filial || "");
      sellerHomeLoaded = false;
      const hello = document.querySelector(".user-chip span:last-child");
      if (hello) hello.textContent = "OlÃ¡, " + String(data.full_name || data.username || "UsuÃ¡rio");
      applyRolePermissions();
      showLoginOverlay(false);
      setStatus("Login realizado");
      if (!appBootstrapped) {
        appBootstrapped = true;
        openPage(isVendedorRole() ? "reservas" : "dashboard");
        loadConfig();
      } else {
        await loadConfig();
        if (isVendedorRole()) openPage("reservas");
      }
    }

    let mfaTempToken = "";

    async function verifyMFALogin(){
      const code = (document.getElementById("login_mfa_code").value || "").trim();
      const errEl = document.getElementById("mfa_login_error");
      if (errEl) errEl.textContent = "";
      
      if (!code || code.length !== 6) {
        if (errEl) errEl.textContent = "Digite o cÃ³digo de 6 dÃ­gitos";
        return;
      }

      setStatus("Verificando cÃ³digo MFA");
      const res = await fetch("/api/app/mfa-login?_cb=" + Date.now(), {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({temp_token: mfaTempToken, code: code})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        const msg = data.message || "CÃ³digo invÃ¡lido ou expirado";
        if (errEl) errEl.textContent = msg;
        setStatus(msg);
        return;
      }

      // Sucesso total
      isAuthenticated = true;
      currentUserRole = normalizeRoleValue(data.role || "");
      window.currentUserRole = currentUserRole;
      window.userPermissions = data.permissions || [];
      currentUserName = String(data.full_name || data.username || "");
      currentUserCPF = String(data.cpf || "");
      currentUserFilial = String(data.branch || data.filial || "");
      const hello = document.querySelector(".user-chip span:last-child");
      if (hello) hello.textContent = "OlÃ¡, " + String(data.full_name || data.username || "UsuÃ¡rio");
      applyRolePermissions();
      showLoginOverlay(false);
      document.getElementById("loginMainFields").classList.remove("hidden");
      document.getElementById("mfaChallengeFields").classList.add("hidden");
      setStatus("Login realizado com MFA");
      if (!appBootstrapped) {
        appBootstrapped = true;
        openPage(isVendedorRole() ? "reservas" : "dashboard");
        loadConfig();
      } else {
        await loadConfig();
        if (isVendedorRole()) openPage("reservas");
      }
    }

    function cancelMFALogin(){
      mfaTempToken = "";
      document.getElementById("loginMainFields").classList.remove("hidden");
      document.getElementById("mfaChallengeFields").classList.add("hidden");
      setStatus("Login cancelado");
    }

    // Fluxo de Setup de MFA
    let currentMFASecret = "";
    let mfaQRCodeInstance = null;

    function openMFASetupModal(){
      document.getElementById("mfaSetupInitial").classList.remove("hidden");
      document.getElementById("mfaSetupStep2").classList.add("hidden");
      document.getElementById("mfaSetupSuccess").classList.add("hidden");
      document.getElementById("mfaSetupModal").classList.remove("hidden");
    }

    function closeMFASetupModal(ev){
      if (ev && ev.target && ev.target.id !== "mfaSetupModal") return;
      document.getElementById("mfaSetupModal").classList.add("hidden");
    }

    async function startMFASetup(){
      setStatus("Gerando segredo MFA");
      const res = await fetch("/api/mfa/setup");
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao iniciar setup MFA");
        return;
      }

      currentMFASecret = data.secret;
      document.getElementById("mfaSecretKey").textContent = data.secret;
      
      const qrEl = document.getElementById("mfaQRCode");
      qrEl.innerHTML = "";
      mfaQRCodeInstance = new QRCode(qrEl, {
        text: data.url,
        width: 180,
        height: 180,
        colorDark: "#000000",
        colorLight: "#ffffff",
        correctLevel: QRCode.CorrectLevel.H
      });

      document.getElementById("mfaSetupInitial").classList.add("hidden");
      document.getElementById("mfaSetupStep2").classList.remove("hidden");
      setStatus("Escaneie o QR Code no seu app");
    }

    async function confirmMFASetup(){
      const code = (document.getElementById("mfaSetupCode").value || "").trim();
      if (!code || code.length !== 6) {
        setStatus("Digite o cÃ³digo de 6 dÃ­gitos");
        return;
      }

      setStatus("Confirmando ativaÃ§Ã£o MFA");
      const res = await fetch("/api/mfa/verify", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({secret: currentMFASecret, code: code})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "CÃ³digo invÃ¡lido");
        return;
      }

      document.getElementById("mfaSetupStep2").classList.add("hidden");
      document.getElementById("mfaSetupSuccess").classList.remove("hidden");
      setStatus("MFA ativado com sucesso");
    }

    function copyMFASecret(){
      const txt = document.getElementById("mfaSecretKey").textContent;
      if (!txt) return;
      navigator.clipboard.writeText(txt).then(() => {
        setStatus("Chave copiada para a Ã¡rea de transferÃªncia");
      });
    }


    async function logoutAppUser(){
      try {
        await fetch("/api/app/logout?_cb=" + Date.now(), {method: "POST"});
      } catch(err) {}
      isAuthenticated = false;
      currentUserRole = "";
      sellerHomeLoaded = false;
      mfaTempToken = ""; 
      window.currentUserRole = "";
      window.userPermissions = [];
      window.location.href = "/"; // Redireciona para a raiz garantindo limpeza de estado
      setStatus("SessÃ£o encerrada");
      const loginErr = document.getElementById("login_error");
      if (loginErr) loginErr.textContent = "";
      document.getElementById("login_username").value = "";
      document.getElementById("login_password").value = "";
      document.getElementById("login_username").focus();
    }

    function maskToken(token){
      const s = String(token || "");
      if (!s) return "";
      if (s.length <= 18) return s;
      return s.slice(0, 8) + "..." + s.slice(-8);
    }

    function authRowById(id){
      return authRows.find((x) => Number(x.id) === Number(id));
    }

    function uiIcon(kind){
      if (kind === "search") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><circle cx=\"11\" cy=\"11\" r=\"7\"/><path d=\"M20 20l-3.5-3.5\"/></svg>";
      }
      if (kind === "add") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M12 5v14\"/><path d=\"M5 12h14\"/></svg>";
      }
      if (kind === "auth") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M9 11V8a3 3 0 0 1 6 0v3\"/><rect x=\"7\" y=\"11\" width=\"10\" height=\"9\" rx=\"2\"/><path d=\"M12 15v2\"/></svg>";
      }
      if (kind === "refresh") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M3 12a9 9 0 0 1 15.5-6.4\"/><path d=\"M20 4v6h-6\"/><path d=\"M21 12a9 9 0 0 1-15.5 6.4\"/><path d=\"M4 20v-6h6\"/></svg>";
      }
      if (kind === "save") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M5 4h12l2 2v14H5z\"/><path d=\"M8 4v6h8V4\"/><path d=\"M9 20v-5h6v5\"/></svg>";
      }
      if (kind === "play") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M8 5l11 7-11 7z\"/></svg>";
      }
      if (kind === "stop") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><rect x=\"7\" y=\"7\" width=\"10\" height=\"10\" rx=\"1\"/></svg>";
      }
      if (kind === "clear") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M3 6h18\"/><path d=\"M8 6V4h8v2\"/><path d=\"M19 6l-1 14H6L5 6\"/><path d=\"M10 11v6\"/><path d=\"M14 11v6\"/></svg>";
      }
      if (kind === "copy") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><rect x=\"9\" y=\"9\" width=\"11\" height=\"11\" rx=\"2\"/><rect x=\"4\" y=\"4\" width=\"11\" height=\"11\" rx=\"2\"/></svg>";
      }
      if (kind === "close") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M6 6l12 12\"/><path d=\"M18 6L6 18\"/></svg>";
      }
      if (kind === "database") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><ellipse cx=\"12\" cy=\"5\" rx=\"7\" ry=\"3\"/><path d=\"M5 5v7c0 1.7 3.1 3 7 3s7-1.3 7-3V5\"/><path d=\"M5 12v7c0 1.7 3.1 3 7 3s7-1.3 7-3v-7\"/></svg>";
      }
      if (kind === "edit") {
        return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M12 20h9\"/><path d=\"M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z\"/></svg>";
      }
      return "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M3 6h18\"/><path d=\"M8 6V4h8v2\"/><path d=\"M19 6l-1 14H6L5 6\"/><path d=\"M10 11v6\"/><path d=\"M14 11v6\"/></svg>";
    }

    function authActionIcon(kind){
      if (kind === "edit") {
        return uiIcon("edit");
      }
      if (kind === "auth") return uiIcon("auth");
      if (kind === "reserve") return uiIcon("play");
      if (kind === "logs") return uiIcon("copy");
      return uiIcon("clear");
    }

    function inferButtonIcon(btn){
      const onclick = String(btn.getAttribute("onclick") || "");
      if (onclick.includes("searchAuthUsers") || onclick.includes("searchReservedCotas") || onclick.includes("searchIDsGrupos") || onclick.includes("searchSolicitacoes") || onclick.includes("searchAppUsers") || onclick.includes("searchModelos") || onclick.includes("searchProdutos") || onclick.includes("searchAssembleias")) return "search";
      if (onclick.includes("openAuthCreateModal") || onclick.includes("prepareAuthCreateForm") || onclick.includes("openIDsGrupoCreateModal") || onclick.includes("openSolicitacaoCreateModal") || onclick.includes("openModeloCreateModal") || onclick.includes("openProdutoCreateModal") || onclick.includes("openAssembleiaCreateModal")) return "add";
      if (onclick.includes("reserveSelectedSolicitacoes")) return "play";
      if (onclick.includes("authenticateSelectedUsers") || onclick.includes("authenticateSelectedUser") || onclick.includes("authenticateUsers")) return "auth";
      if (onclick.includes("deleteSelectedReservedCotas") || onclick.includes("deleteReservedCota") || onclick.includes("deleteSelectedIDsGrupos") || onclick.includes("deleteIDsGrupo") || onclick.includes("deleteModelo") || onclick.includes("deleteProduto") || onclick.includes("deleteAssembleia")) return "clear";
      if (onclick.includes("openIDsGrupoEditModal")) return "edit";
      if (onclick.includes("createDatabaseTables")) return "database";
      if (onclick.includes("clearDatabaseTables")) return "clear";
      if (onclick.includes("dropDatabaseTables")) return "clear";
      if (onclick.includes("loadConfig")) return "refresh";
      if (onclick.includes("saveConfig") || onclick.includes("saveAuthUser") || onclick.includes("saveIDsGrupo") || onclick.includes("saveModelo") || onclick.includes("saveProduto") || onclick.includes("saveAssembleia")) return "save";
      if (onclick.includes("runEngine")) return "play";
      if (onclick.includes("stopEngine")) return "stop";
      if (onclick.includes("clearLogs")) return "clear";
      if (onclick.includes("close")) return "close";

      const text = String(btn.textContent || "").toLowerCase();
      if (text.includes("buscar")) return "search";
      if (text.includes("novo")) return "add";
      if (text.includes("autent")) return "auth";
      if (text.includes("salvar")) return "save";
      if (text.includes("executar")) return "play";
      if (text.includes("parar")) return "stop";
      if (text.includes("limpar")) return "clear";
      if (text.includes("deletar")) return "clear";
      if (text.includes("atualizar")) return "refresh";
      if (text.includes("fechar")) return "close";
      return "";
    }

    function decorateButtons(){
      const all = document.querySelectorAll("button.btn");
      for (const btn of all) {
        if (btn.dataset.iconified === "1") continue;
        const kind = btn.dataset.icon || inferButtonIcon(btn);
        if (!kind) continue;
        const label = String(btn.innerHTML || "").trim();
        btn.innerHTML = "<span class=\"btn-ico\" aria-hidden=\"true\">" + uiIcon(kind) + "</span><span class=\"btn-label\">" + label + "</span>";
        btn.dataset.iconified = "1";

        if (kind === "stop" || kind === "clear") {
          btn.classList.add("btn-danger");
        }
      }
    }

    function renderAuthTable(items){
      authRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("authTableBody");
      const selectAll = document.getElementById("auth_select_all");
      if (selectAll) selectAll.checked = false;

      if (!authRows.length) {
        body.innerHTML = "<tr><td colspan=\"8\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      const rows = authRows.map((u) => {
        const id = Number(u.id || 0);
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"auth-row-check\" value=\"" + String(id) + "\" /></td>" +
          "<td>" + escapeHtml(id) + "</td>" +
          "<td>" + escapeHtml(u.company_code || "") + "</td>" +
          "<td>" + escapeHtml(u.cpf || "") + "</td>" +
          "<td>" + escapeHtml(u.account_user || "") + "</td>" +
          "<td>" + escapeHtml(u.account_password || "") + "</td>" +
          "<td>" + escapeHtml(maskToken(u.token || "")) + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openAuthEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Autenticar\" aria-label=\"Autenticar\" onclick=\"authenticateAuthUser(" + String(id) + ")\">" + authActionIcon("auth") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteAuthUser(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
      body.innerHTML = rows;
    }

    function normalizeYesNoValue(raw){
      const original = decodeMojibake(String(raw || "").trim());
      if (!original) return "N\u00e3o";
      const lower = original.toLowerCase();
      const norm = (lower.normalize ? lower.normalize("NFD").replace(/[\u0300-\u036f]/g, "") : lower).trim();
      if (norm === "sim" || norm === "yes" || norm === "true" || norm === "1") return "Sim";
      if (norm === "nao" || norm === "no" || norm === "false" || norm === "0") return "N\u00e3o";
      return "N\u00e3o";
    }

    function clearSolicitacaoForm(){
      for (const field of solicitacaoFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const plan = document.getElementById("sol_plano");
      if (plan) {
        plan.value = "N\u00e3o";
        if (plan.value !== "N\u00e3o") plan.value = "Nao";
        if (!plan.value && plan.options && plan.options.length > 0) plan.selectedIndex = 0;
      }
      const restricao = document.getElementById("sol_com_restricao");
      if (restricao) {
        restricao.value = "N\u00e3o";
        if (restricao.value !== "N\u00e3o") restricao.value = "Nao";
        if (!restricao.value && restricao.options && restricao.options.length > 0) restricao.selectedIndex = 0;
      }
      const dt = document.getElementById("sol_data_hora_solicitacao");
      if (dt) dt.value = localDateTimeISOSeconds();
    }

    function fillSolicitacaoForm(data){
      if (!data) {
        clearSolicitacaoForm();
        return;
      }
      document.getElementById("sol_id").value = data.id || "";
      document.getElementById("sol_data_hora_solicitacao").value = data.requested_at || "";
      document.getElementById("sol_filial").value = data.branch || "";
      document.getElementById("sol_vendedor").value = data.seller_name || "";
      document.getElementById("sol_cpf").value = data.cpf || "";
      document.getElementById("sol_modelo").value = data.model_name || "";
      document.getElementById("sol_plano").value = normalizeYesNoValue(data.plan || "");
      document.getElementById("sol_qtde_parcelas").value = data.installments || "";
      document.getElementById("sol_perc_lance").value = formatPercentInputValue(data.bid_percent || "");
      document.getElementById("sol_com_restricao").value = normalizeYesNoValue(data.with_restriction || "");
      document.getElementById("sol_grupo").value = data.group_code || "";
      document.getElementById("sol_observacao").value = data.notes || "";
      document.getElementById("sol_id_cota").value = data.requested_quota_id || "";
      document.getElementById("sol_grupo_atendido").value = data.served_group || "";
      document.getElementById("sol_cota_r_d").value = data.quota_rd || "";
      document.getElementById("sol_data_hora_atendimento").value = formatDateTimeBR(data.served_at || "");
      document.getElementById("sol_situacao").value = data.status || "";
      document.getElementById("sol_lance_contemplacao").value = data.contemplation_bid || "";
      syncSolicitacaoParcelasByGrupo();
    }

    function gatherSolicitacaoForm(){
      return {
        id: Number(document.getElementById("sol_id").value || 0),
        requested_at: document.getElementById("sol_data_hora_solicitacao").value || "",
        branch: document.getElementById("sol_filial").value || "",
        seller_name: document.getElementById("sol_vendedor").value || "",
        cpf: (document.getElementById("sol_cpf").value || "").replace(/\D/g, ""),
        model_name: document.getElementById("sol_modelo").value || "",
        plan: document.getElementById("sol_plano").value || "",
        installments: document.getElementById("sol_qtde_parcelas").value || "",
        bid_percent: percentInputToRaw(document.getElementById("sol_perc_lance").value || ""),
        with_restriction: document.getElementById("sol_com_restricao").value || "",
        group_code: document.getElementById("sol_grupo").value || "",
        notes: document.getElementById("sol_observacao").value || "",
        requested_quota_id: document.getElementById("sol_id_cota").value || "",
        served_group: document.getElementById("sol_grupo_atendido").value || "",
        quota_rd: document.getElementById("sol_cota_r_d").value || "",
        served_at: document.getElementById("sol_data_hora_atendimento").value || "",
        status: document.getElementById("sol_situacao").value || "",
        contemplation_bid: document.getElementById("sol_lance_contemplacao").value || ""
      };
    }

    function localDateTimeISOSeconds(){
      const d = new Date();
      const yyyy = d.getFullYear();
      const mm = String(d.getMonth() + 1).padStart(2, "0");
      const dd = String(d.getDate()).padStart(2, "0");
      const hh = String(d.getHours()).padStart(2, "0");
      const mi = String(d.getMinutes()).padStart(2, "0");
      const ss = String(d.getSeconds()).padStart(2, "0");
      return yyyy + "-" + mm + "-" + dd + "T" + hh + ":" + mi + ":" + ss;
    }

    function clearSolicitarCotaForm(){
      for (const field of solicitarFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const plan = document.getElementById("request_plano");
      if (plan) {
        plan.value = "N\u00e3o";
        if (plan.value !== "N\u00e3o") plan.value = "Nao";
        if (!plan.value && plan.options && plan.options.length > 0) plan.selectedIndex = 0;
      }
      const restricao = document.getElementById("request_com_restricao");
      if (restricao) {
        restricao.value = "N\u00e3o";
        if (restricao.value !== "N\u00e3o") restricao.value = "Nao";
        if (!restricao.value && restricao.options && restricao.options.length > 0) restricao.selectedIndex = 0;
      }
      const dt = document.getElementById("request_data_hora_solicitacao");
      if (dt) dt.value = localDateTimeISOSeconds();
      const nome = document.getElementById("request_vendedor");
      if (nome) nome.value = currentUserName || "";
      const cpf = document.getElementById("request_cpf");
      if (cpf) cpf.value = currentUserCPF || "";
      const branch = document.getElementById("request_filial");
      if (branch && currentUserFilial) branch.value = currentUserFilial;
    }

    function percentInputToRaw(value){
      let s = String(value || "").trim();
      if (!s) return "";
      s = s.replace(/%/g, "").trim();
      if (s.includes(",")) {
        s = s.replace(/\./g, "").replace(",", ".");
      }
      const n = Number(s);
      if (!Number.isFinite(n)) return "";
      return String(n);
    }

    function formatPercentInputValue(value){
      const raw = percentInputToRaw(value);
      if (!raw) return "";
      const n = Number(raw);
      if (!Number.isFinite(n)) return "";
      return n.toLocaleString("pt-BR", {minimumFractionDigits: 2, maximumFractionDigits: 2}) + "%";
    }

    async function loadSolicitarModeloOptions(selected, selectId, emptyLabel){
      const targetId = String(selectId || "request_modelo");
      const select = document.getElementById(targetId);
      if (!select) return;
      const selectedValue = String(selected !== undefined ? selected : (select.value || ""));
      const placeholder = String(emptyLabel || "Selecione o modelo");
      const normKey = (v) => String(v || "").trim().toLowerCase();
      try {
        const res = await fetch("/api/models?q=");
        const data = await res.json().catch(() => ({}));
        if (!res.ok || data.ok === false) {
          setStatus((data && data.message) ? data.message : "Falha ao carregar modelos");
          return;
        }
        const seen = new Set();
        const nomes = [];
        const items = Array.isArray(data.items) ? data.items : [];
        for (const item of items) {
          const nome = String((item && item.model_name) || "").trim();
          if (!nome) continue;
          const key = normKey(nome);
          if (seen.has(key)) continue;
          seen.add(key);
          nomes.push(nome);
        }
        nomes.sort((a, b) => String(a).localeCompare(String(b)));

        select.innerHTML = "";
        select.appendChild(new Option(placeholder, ""));
        for (const nome of nomes) {
          select.appendChild(new Option(nome, nome));
        }
        if (selectedValue) {
          const exists = nomes.some((n) => normKey(n) === normKey(selectedValue));
          if (!exists) select.appendChild(new Option(selectedValue, selectedValue));
          select.value = selectedValue;
        } else {
          select.value = "";
        }
        if (!nomes.length) {
          setStatus("Nenhum modelo encontrado");
        }
      } catch (_) {
        setStatus("Falha ao carregar modelos");
      }
    }

    let requestGrupoParcelasFetchToken = 0;
    let requestGrupoDebounceTimer = null;
    async function syncSolicitarParcelasByGrupo(forceLookup){
      const grupoEl = document.getElementById("request_grupo");
      const parcelasEl = document.getElementById("request_qtde_parcelas");
      const percEl = document.getElementById("request_perc_lance");
      if (!grupoEl || !parcelasEl || !percEl) return;
      const raw = String(grupoEl.value || "").trim();
      const onlyDigits = raw.replace(/\D+/g, "");
      if (!onlyDigits) {
        parcelasEl.value = "";
        percEl.value = "";
        return;
      }
      if (!forceLookup && onlyDigits.length < 4) return;
      const currentToken = ++requestGrupoParcelasFetchToken;
      try {
        const [parcelasResult, percResult] = await Promise.allSettled([
          fetch("/api/active_groups/parcelas?group_code=" + encodeURIComponent(onlyDigits)),
          fetch("/api/assembleias/perclance?group_code=" + encodeURIComponent(onlyDigits))
        ]);
        if (currentToken !== requestGrupoParcelasFetchToken) return;

        if (parcelasResult.status === "fulfilled") {
          const parcelasRes = parcelasResult.value;
          let data = null;
          try { data = await parcelasRes.json(); } catch (_) {}
          if (parcelasRes.ok && data && data.ok !== false && data.found) {
            const parcelas = Number(data.parcelas || 0);
            parcelasEl.value = parcelas > 0 ? String(parcelas) : "";
          } else if (!parcelasRes.ok && data && data.message) {
            setStatus("Parcelas: " + data.message);
          }
        }

        if (percResult.status === "fulfilled") {
          const percRes = percResult.value;
          let percData = null;
          try { percData = await percRes.json(); } catch (_) {}
          if (percRes.ok && percData && percData.ok !== false && percData.found) {
            percEl.value = formatPercentInputValue(percData.bid_percent || "");
          } else if (!percRes.ok && percData && percData.message) {
            setStatus("Perc. lance: " + percData.message);
          }
        }
      } catch (_) {
        // Preserva valor atual em caso de falha temporaria.
      }
    }

    let solicitacaoGrupoParcelasFetchToken = 0;
    let solicitacaoGrupoDebounceTimer = null;
    async function syncSolicitacaoParcelasByGrupo(forceLookup){
      const grupoEl = document.getElementById("sol_grupo");
      const parcelasEl = document.getElementById("sol_qtde_parcelas");
      const percEl = document.getElementById("sol_perc_lance");
      if (!grupoEl || !parcelasEl || !percEl) return;
      const raw = String(grupoEl.value || "").trim();
      const onlyDigits = raw.replace(/\D+/g, "");
      if (!onlyDigits) {
        parcelasEl.value = "";
        percEl.value = "";
        return;
      }
      if (!forceLookup && onlyDigits.length < 4) return;
      const currentToken = ++solicitacaoGrupoParcelasFetchToken;
      try {
        const [parcelasResult, percResult] = await Promise.allSettled([
          fetch("/api/active_groups/parcelas?group_code=" + encodeURIComponent(onlyDigits)),
          fetch("/api/assembleias/perclance?group_code=" + encodeURIComponent(onlyDigits))
        ]);
        if (currentToken !== solicitacaoGrupoParcelasFetchToken) return;

        if (parcelasResult.status === "fulfilled") {
          const parcelasRes = parcelasResult.value;
          let data = null;
          try { data = await parcelasRes.json(); } catch (_) {}
          if (parcelasRes.ok && data && data.ok !== false && data.found) {
            const parcelas = Number(data.parcelas || 0);
            parcelasEl.value = parcelas > 0 ? String(parcelas) : "";
          } else if (!parcelasRes.ok && data && data.message) {
            setStatus("Parcelas: " + data.message);
          }
        }

        if (percResult.status === "fulfilled") {
          const percRes = percResult.value;
          let percData = null;
          try { percData = await percRes.json(); } catch (_) {}
          if (percRes.ok && percData && percData.ok !== false && percData.found) {
            percEl.value = formatPercentInputValue(percData.bid_percent || "");
          } else if (!percRes.ok && percData && percData.message) {
            setStatus("Perc. lance: " + percData.message);
          }
        }
      } catch (_) {
        // Preserva valor atual em caso de falha temporaria.
      }
    }

    function gatherSolicitarCotaForm(){
      return {
        id: 0,
        requested_at: document.getElementById("request_data_hora_solicitacao").value || localDateTimeISOSeconds(),
        branch: document.getElementById("request_filial").value || "",
        seller_name: document.getElementById("request_vendedor").value || "",
        cpf: (document.getElementById("request_cpf").value || "").replace(/\D/g, ""),
        model_name: document.getElementById("request_modelo").value || "",
        plan: document.getElementById("request_plano").value || "",
        installments: document.getElementById("request_qtde_parcelas").value || "",
        bid_percent: percentInputToRaw(document.getElementById("request_perc_lance").value || ""),
        with_restriction: document.getElementById("request_com_restricao").value || "",
        group_code: document.getElementById("request_grupo").value || "",
        notes: document.getElementById("request_observacao").value || "",
        requested_quota_id: "",
        served_group: "",
        quota_rd: "",
        served_at: "",
        status: "",
        contemplation_bid: ""
      };
    }

    async function saveSolicitarCota(){
      const payload = gatherSolicitarCotaForm();
      if (!String(payload.branch || "").trim()) {
        setStatus("Preencha o campo filial");
        return;
      }
      if (!String(payload.seller_name || "").trim()) {
        setStatus("Preencha o campo Vendedor");
        return;
      }
      if (!String(payload.cpf || "").trim()) {
        setStatus("Preencha o campo CPF");
        return;
      }
      if (!String(payload.model_name || "").trim()) {
        setStatus("Preencha o campo Modelo");
        return;
      }
      setStatus("Salvando solicitaÃ§Ã£o");
      const res = await fetch("/api/solicitacoes/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar solicitaÃ§Ã£o");
        return;
      }
      setStatus(data.message || "SolicitaÃ§Ã£o salva");
      clearSolicitarCotaForm();
      if (normalizeRoleValue(currentUserRole) !== "vendedor") {
        await searchSolicitacoes();
      }
    }

    async function autoFillSolicitacaoByCPF(){
      const rawCpf = document.getElementById("request_cpf").value || "";
      const cpf = rawCpf.replace(/\D/g, "");
      if (cpf.length !== 11 && cpf.length !== 14) {
          return;
      }
      
      console.log("[MFA-Request] Iniciando busca por CPF:", cpf);
      setStatus("Buscando dados por CPF...");
      try {
        const res = await fetch("/api/solicitacoes/last-by-cpf?cpf=" + encodeURIComponent(cpf));
        const data = await res.json();
        console.log("[MFA-Request] Resposta:", data);

        if (res.ok && data.ok !== false && data.found && data.item) {
          const item = data.item;
          const setField = (id, val) => {
              const el = document.getElementById(id);
              if (el) el.value = val || "";
          };

          try { setField("request_filial", item.branch); } catch(e) {}
          try { setField("request_vendedor", item.seller_name); } catch(e) {}
          
          // Modelo removido do auto-preenchimento por CPF conforme solicitado (escolha manual)
          
          try { setField("request_plano", normalizeYesNoValue(item.plan)); } catch(e) {}
          try { setField("request_com_restricao", normalizeYesNoValue(item.with_restriction)); } catch(e) {}
          try { setField("request_observacao", item.notes); } catch(e) {}
          
          setStatus("Dados preenchidos via CPF");
          console.log("[MFA-Request] Sucesso para o CPF:", cpf);
        } else {
          setStatus("Nenhum registro anterior para este CPF");
          console.log("[MFA-Request] NÃ£o encontrado para o CPF:", cpf);
        }
      } catch (e) {
        console.error("[MFA-Request] Erro fatal:", e);
        setStatus("Erro ao buscar dados por CPF");
      }
    }

    async function autoFillSolicitacaoEditByCPF(){
      const id = document.getElementById("sol_id").value;
      if (id && id !== "0" && id !== "") {
        console.log("[MFA] Ignorando preenchimento automÃ¡tico pois o ID jÃ¡ estÃ¡ preenchido (EdiÃ§Ã£o).");
        return;
      }

      const rawCpf = document.getElementById("sol_cpf").value || "";
      const cpf = rawCpf.replace(/\D/g, "");
      if (cpf.length !== 11 && cpf.length !== 14) {
          return;
      }
      
      console.log("[MFA] Iniciando busca por CPF:", cpf);
      setStatus("Buscando dados por CPF...");
      try {
        const res = await fetch("/api/solicitacoes/last-by-cpf?cpf=" + encodeURIComponent(cpf));
        const data = await res.json();
        console.log("[MFA] Resposta da API:", data);

        if (res.ok && data.ok !== false && data.found && data.item) {
          const item = data.item;
          
          const setField = (id, val) => {
              const el = document.getElementById(id);
              if (el) el.value = val || "";
          };

          try { setField("sol_filial", item.branch); } catch(e) {}
          try { setField("sol_vendedor", item.seller_name); } catch(e) {}
          
          // Modelo removido do auto-preenchimento por CPF conforme solicitado (escolha manual)
          
          try { setField("sol_plano", normalizeYesNoValue(item.plan)); } catch(e) {}
          try { setField("sol_com_restricao", normalizeYesNoValue(item.with_restriction)); } catch(e) {}
          try { setField("sol_observacao", item.notes); } catch(e) {}
          
          setStatus("Dados preenchidos via CPF");
          console.log("[MFA] Campos preenchidos com sucesso para o CPF:", cpf);
        } else {
          setStatus("Nenhum registro anterior para este CPF");
          console.log("[MFA] Nenhum registro encontrado para o CPF:", cpf);
        }
      } catch (e) {
        console.error("[MFA] Erro fatal no autoFillSolicitacaoEditByCPF:", e);
        setStatus("Erro ao buscar dados por CPF");
      }
    }

    async function autoFillSolicitacaoByGroup(prefix){
      const groupEl = document.getElementById(prefix + "_grupo");
      if (!groupEl) return;
      const code = groupEl.value.replace(/\D/g, "");
      if (!code) return;

      console.log("[MFA-Group] Buscando dados para grupo:", code);
      setStatus("Validando grupo...");
      try {
        const res = await fetch("/api/available_group_ids/check?code=" + encodeURIComponent(code));
        const data = await res.json();
        console.log("[MFA-Group] Resposta:", data);

        if (res.ok && data.ok !== false && data.found && data.item) {
          const item = data.item;
          const installmentsEl = document.getElementById(prefix + "_qtde_parcelas");
          if (installmentsEl) installmentsEl.value = "";

          // Prioriza o mesmo calculo exibido em Grupos Ativos.
          // Fallback para prazo legado apenas se o endpoint calculado nao retornar.
          try {
            const parcelasRes = await fetch("/api/active_groups/parcelas?group_code=" + encodeURIComponent(code));
            const parcelasData = await parcelasRes.json();
            if (parcelasRes.ok && parcelasData && parcelasData.ok !== false && parcelasData.found) {
              const parcelasCalc = Number(parcelasData.parcelas || 0);
              if (installmentsEl) installmentsEl.value = parcelasCalc > 0 ? String(parcelasCalc) : "";
            } else if (installmentsEl) {
              installmentsEl.value = item.Prazo || "";
            }
          } catch (_) {
            if (installmentsEl) installmentsEl.value = item.Prazo || "";
          }
          
          // Novo: Buscar percentual de lance da assembleia para este grupo
          try {
              const pRes = await fetch("/api/assembleias/perclance?group_code=" + encodeURIComponent(code));
              const pData = await pRes.json();
              if (pRes.ok && pData && pData.ok !== false && pData.found) {
                  const percEl = document.getElementById(prefix + "_perc_lance");
                  if (percEl) percEl.value = formatPercentInputValue(pData.bid_percent || "");
              }
          } catch(e) { console.error("Erro ao buscar percentual de lance:", e); }

          setStatus("Dados do grupo carregados");
          console.log("[MFA-Group] Sucesso para o grupo:", code);
        } else {
          setStatus("Grupo nÃ£o encontrado ou indisponÃ­vel");
          console.log("[MFA-Group] Grupo invÃ¡lido ou indisponÃ­vel:", code);
        }
      } catch (e) {
        console.error("[MFA-Group] Erro fatal:", e);
        setStatus("Erro ao validar grupo");
      }
    }

    function buildSuccessMessage(item){
      const nome = String(item.nome || "").trim() || "client_name";
      const branch = String(item.branch || "").trim().toUpperCase();
      const group_code = String(item.served_group || "").trim() || "-";
      const quota_rd = String(item.cota_rd || "").trim() || "-";
      const model_name = String(item.model_name || "").trim() || "-";
      const solicitada = String(item.requested_at || "").trim();
      const atendida = String(item.served_at || "").trim();
      const sla = String(item.sla || "").trim();
      const lines = [
        "Ol\u00e1, " + nome + (branch ? " - " + branch : ""),
        "Sua solicita\u00e7\u00e3o foi atendida na seguinte condi\u00e7\u00e3o:"
      ];
      if (branch !== "TER") {
        lines.push("Grupo: " + group_code);
      }
      lines.push("Cota-R-D: " + quota_rd);
      lines.push("Modelo: " + model_name);
      if (solicitada) lines.push("Solicitada: " + solicitada);
      if (atendida) lines.push("Atendida: " + atendida);
      if (sla) lines.push("SLA: " + sla);
      return lines.join("\n");
    }

    function renderSuccessMessagesModal(){
      const list = document.getElementById("successMessagesList");
      if (!list) return;
      const head = "<div class=\"success-msg-row success-msg-head\">" +
        "<div><input id=\"success_msg_select_all\" type=\"checkbox\" onclick=\"toggleSelectAllSuccessMessages()\" /></div>" +
        "<div>message</div>" +
      "</div>";
      if (!successMessageItems.length) {
        list.innerHTML = head + "<div class=\"success-msg-row\"><div></div><div>Nenhuma solicita\u00e7\u00e3o atendida nesta execu\u00e7\u00e3o.</div></div>";
        return;
      }
      const rows = successMessageItems.map((item, idx) => {
        const msg = buildSuccessMessage(item);
        return "<div class=\"success-msg-row\">" +
          "<div><input type=\"checkbox\" class=\"success-msg-check\" value=\"" + String(idx) + "\" checked /></div>" +
          "<div><pre class=\"success-msg-text\">" + escapeHtml(msg) + "</pre></div>" +
        "</div>";
      }).join("");
      list.innerHTML = head + rows;
    }

    function openSuccessMessagesModal(items){
      successMessageItems = Array.isArray(items) ? items : [];
      renderSuccessMessagesModal();
      document.getElementById("successMessagesModal").classList.remove("hidden");
    }

    function closeSuccessMessagesModal(ev){
      if (ev && ev.target && ev.target.id !== "successMessagesModal") return;
      document.getElementById("successMessagesModal").classList.add("hidden");
    }

    function toggleSelectAllSuccessMessages(){
      const all = !!document.getElementById("success_msg_select_all").checked;
      const checks = document.querySelectorAll(".success-msg-check");
      for (const el of checks) el.checked = all;
    }

    async function copySuccessMessages(){
      const selected = [];
      const checks = document.querySelectorAll(".success-msg-check:checked");
      for (const el of checks) {
        const idx = Number(el.value || -1);
        if (idx >= 0 && idx < successMessageItems.length) {
          selected.push(successMessageItems[idx]);
        }
      }
      const items = selected.length ? selected : successMessageItems;
      if (!items.length) {
        setStatus("Nenhuma message para copiar");
        return;
      }
      const text = items.map(buildSuccessMessage).join("\n\n");
      try {
        await navigator.clipboard.writeText(text);
      } catch (err) {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
      }
      setStatus("Mensagens copiadas");
    }

    function selectedSolicitacaoIDs(){
      const out = [];
      const checks = document.querySelectorAll(".sol-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllSolicitacoes(){
      const all = !!document.getElementById("sol_select_all").checked;
      const checks = document.querySelectorAll(".sol-row-check");
      for (const el of checks) el.checked = all;
    }

    function reserveSelectedSolicitacoes(){
      const ids = selectedSolicitacaoIDs();
      if (!ids.length) {
        setStatus("Selecione ao menos uma solicitacao");
        return;
      }
      (async () => {
        setStatus("Reservando solicitaÃ§Ãµes selecionadas");
        const res = await fetch("/api/solicitacoes/reservar-batch", {
          method: "POST",
          headers: {"Content-Type":"application/json"},
          body: JSON.stringify({ids})
        });
        const data = await res.json();
        if (!res.ok || data.ok === false) {
          setStatus(data.message || "Falha ao reservar selecionadas");
          return;
        }
        setStatus(data.message || "Reserva em lote concluÃ­da");
        if (Array.isArray(data.success_items) && data.success_items.length > 0) {
          openSuccessMessagesModal(data.success_items);
        }
        await searchSolicitacoes();
      })();
    }

    async function openAuthModalByCPF(cpf){
      const cleanCPF = String(cpf || "").trim();
      openPage("config");
      openConfigSection("users");
      const search = document.getElementById("auth_search");
      if (search) search.value = cleanCPF;
      if (!cleanCPF) {
        await searchAuthUsers();
        return;
      }
      try {
        const res = await fetch("/api/auth/user/find?q=" + encodeURIComponent(cleanCPF));
        const data = await res.json();
        if (!res.ok || data.ok === false) {
          setStatus("CPF sem token: abra o usuÃ¡rio e autentique");
          await searchAuthUsers();
          return;
        }
        fillAuthForm(data);
        document.getElementById("authModalTitle").textContent = "Editar UsuÃ¡rio";
        document.getElementById("authEditModal").classList.remove("hidden");
        setStatus("UsuÃ¡rio carregado para autenticaÃ§Ã£o");
      } catch (err) {
        await searchAuthUsers();
      }
    }

    function reserveSolicitacao(id, cpf){
      if (!id) return;
      (async () => {
        setStatus("Reservando solicitaÃ§Ã£o");
        const res = await fetch("/api/solicitacoes/reservar", {
          method: "POST",
          headers: {"Content-Type":"application/json"},
          body: JSON.stringify({id})
        });
        const data = await res.json();
        if (data && data.code === "no_token") {
          setStatus(data.message || "CPF sem token is_active na auth");
          await openAuthModalByCPF(data.cpf || cpf || "");
          return;
        }
        if (!res.ok || data.ok === false) {
          setStatus(data.message || "Falha na reserva");
          return;
        }
        setStatus(data.message || "SolicitaÃ§Ã£o atendida");
        if (Array.isArray(data.success_items) && data.success_items.length > 0) {
          openSuccessMessagesModal(data.success_items);
        }
        await searchSolicitacoes();
      })();
    }

    function openSolicitacaoCreateModal(){
      clearSolicitacaoForm();
      const t = document.getElementById("solicitacaoModalTitle");
      if (t) t.textContent = "Nova SolicitaÃ§Ã£o";
      loadSolicitarModeloOptions("", "sol_modelo", "Selecione o modelo");
      document.getElementById("solicitacaoEditModal").classList.remove("hidden");
    }

    function closeSolicitacaoEditModal(ev){
      if (ev && ev.target && ev.target.id !== "solicitacaoEditModal") return;
      document.getElementById("solicitacaoEditModal").classList.add("hidden");
    }

    async function openSolicitacaoEditModal(id){
      setStatus("Carregando solicitaÃ§Ã£o");
      const t = document.getElementById("solicitacaoModalTitle");
      if (t) t.textContent = "Editar SolicitaÃ§Ã£o";
      const res = await fetch("/api/solicitacoes/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar solicitaÃ§Ã£o");
        return;
      }
      await loadSolicitarModeloOptions(data.model_name || "", "sol_modelo", "Selecione o modelo");
      fillSolicitacaoForm(data);
      document.getElementById("solicitacaoEditModal").classList.remove("hidden");
      setStatus("SolicitaÃ§Ã£o carregada");
    }

    function renderSolicitacoesTable(items){
      solicitacaoRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("solicitacaoTableBody");
      const selectAll = document.getElementById("sol_select_all");
      if (selectAll) selectAll.checked = false;
      if (!solicitacaoRows.length) {
        body.innerHTML = "<tr><td colspan=\"17\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      const rows = solicitacaoRows.map((s) => {
        const id = Number(s.id || 0);
        const notes = String(s.notes || "");
        const grAten = String(s.served_group || "").trim();
        const quota_rd = String(s.quota_rd || "").trim();
        let grupoCotaRD = "";
        if (grAten && quota_rd) grupoCotaRD = grAten + "-" + quota_rd;
        else if (grAten) grupoCotaRD = grAten;
        else if (quota_rd) grupoCotaRD = quota_rd;
        const isAttended = String(s.status || "").toLowerCase().includes("atendid") || (grAten !== "" && quota_rd !== "");
        const qtdNum = Number(String(s.qtd_solicitada || "").replace(",", "."));
        const qtdAlert = Number.isFinite(qtdNum) && qtdNum >= 10;
        const reserveBtnAttr = isAttended ? "disabled title=\"SolicitaÃ§Ã£o jÃ¡ atendida\"" : "title=\"Reservar\"";
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"sol-row-check\" value=\"" + String(id) + "\" " + (isAttended ? "disabled" : "") + " /></td>" +
          "<td>" + escapeHtml(id) + "</td>" +
          "<td>" + escapeHtml(s.branch || "") + "</td>" +
          "<td>" + escapeHtml(s.seller_name || "") + "</td>" +
          "<td>" + escapeHtml(s.cpf || "") + "</td>" +
          "<td>" + escapeHtml(s.model_name || "") + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(s.requested_at || "")) + "</td>" +
          "<td>" + escapeHtml(s.plan || "") + "</td>" +
          "<td>" + escapeHtml(s.installments || "") + "</td>" +
          "<td>" + escapeHtml(formatPercent(s.bid_percent || "")) + "</td>" +
          "<td>" + escapeHtml(s.with_restriction || "") + "</td>" +
          "<td>" + escapeHtml(s.group_code || "") + "</td>" +
          "<td" + (qtdAlert ? " style=\"color:#C62127;font-weight:700;\"" : "") + ">" + escapeHtml(s.qtd_solicitada || "") + "</td>" +
          "<td title=\"" + escapeHtml(notes) + "\"><span class=\"sol-cell-ellipsis\">" + escapeHtml(notes) + "</span></td>" +
          "<td>" + escapeHtml(s.requested_quota_id || "") + "</td>" +
          "<td>" + escapeHtml(grupoCotaRD) + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openSolicitacaoEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-success\" " + reserveBtnAttr + " aria-label=\"Reservar\" onclick=\"reserveSolicitacao(" + String(id) + ", &quot;" + escapeHtml(s.cpf || "") + "&quot;)\">" + authActionIcon("reserve") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteSolicitacao(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
      body.innerHTML = rows;
    }

        async function searchSolicitacoes(){
      const q = (document.getElementById("solicitacao_search").value || "").trim();
      const column = (document.getElementById("sol_search_column").value || "").trim();
      const status = (document.getElementById("sol_status_filter").value || "pending").trim();
      const from = (document.getElementById("sol_from").value || "").trim();
      const to = (document.getElementById("sol_to").value || "").trim();
      setStatus("Buscando solicitaÃ§Ãµes");
      const res = await fetch("/api/solicitacoes?q=" + encodeURIComponent(q) + "&column=" + encodeURIComponent(column) + "&status=" + encodeURIComponent(status) + "&from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro na busca");
        renderSolicitacoesTable([]);
        return;
      }
      renderSolicitacoesTable(data.items || []);
      setStatus("SolicitaÃ§Ãµes carregadas: " + String(data.count || 0));
    }

    function parseDateTimeFlexible(value){
      const s = String(value || "").trim();
      if (!s) return null;
      const m = s.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?/);
      if (m) {
        const sec = Number(m[6] || "0");
        const dt = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]), Number(m[4]), Number(m[5]), sec);
        if (!Number.isNaN(dt.getTime())) return dt;
      }
      const d = new Date(s);
      if (!Number.isNaN(d.getTime())) return d;
      return null;
    }

    function formatSLAForMinhaSolicitacao(s){
      const start = parseDateTimeFlexible(s.requested_at || "");
      if (!start) return "";
      const endAtendida = parseDateTimeFlexible(s.served_at || "");
      const end = endAtendida || new Date();
      const diffMs = end.getTime() - start.getTime();
      if (!Number.isFinite(diffMs) || diffMs < 0) return "";
      const totalMin = Math.floor(diffMs / 60000);
      const dd = Math.floor(totalMin / (24 * 60));
      const hh = Math.floor((totalMin % (24 * 60)) / 60);
      const mm = totalMin % 60;
      if (dd > 0) return String(dd) + "d " + String(hh) + "h " + String(mm) + "m";
      if (hh > 0) return String(hh) + "h " + String(mm) + "m";
      return String(mm) + "m";
    }

    function normalizeMinhaSituacaoLabel(raw){
      const s = String(raw || "").trim().toLowerCase();
      if (!s) return "Solicitada";
      if (s.startsWith("solicit")) return "Solicitada";
      if (s.startsWith("atendid")) return "Atendida";
      if (s.startsWith("digit")) return "Digitada";
      if (s.startsWith("expir")) return "Expirada";
      return raw;
    }

    function grupoCotaRDText(s){
      const group_code = String(s.served_group || s.group_code || "").trim();
      const quota_rd = String(s.quota_rd || "").trim();
      if (group_code && quota_rd) return group_code + "-" + quota_rd;
      if (group_code) return group_code;
      if (quota_rd) return quota_rd;
      return "-";
    }
    function renderMinhasSolicitacoesTable(items){
      minhasSolicitacaoRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("mySolicitacaoTableBody");
      if (!body) return;
      if (!minhasSolicitacaoRows.length) {
        body.innerHTML = "<tr><td colspan=\"9\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = minhasSolicitacaoRows.map((s) => {
        return "<tr>" +
          "<td>" + escapeHtml(String(s.id || "")) + "</td>" +
          "<td>" + escapeHtml(s.branch || "") + "</td>" +
          "<td>" + escapeHtml(s.seller_name || "") + "</td>" +
          "<td>" + escapeHtml(s.model_name || "") + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(s.requested_at || "")) + "</td>" +
          "<td>" + escapeHtml(grupoCotaRDText(s)) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(s.served_at || "")) + "</td>" +
          "<td>" + escapeHtml(formatSLAForMinhaSolicitacao(s)) + "</td>" +
          "<td>" + escapeHtml(normalizeMinhaSituacaoLabel(s.status || "")) + "</td>" +
        "</tr>";
      }).join("");
    }

    async function searchMinhasSolicitacoes(){
      const q = (document.getElementById("my_solicitacao_search")?.value || "").trim();
      const status = (document.getElementById("my_sol_status_filter")?.value || "atendida").trim().toLowerCase();
      const from = (document.getElementById("my_sol_from")?.value || "").trim();
      const to = (document.getElementById("my_sol_to")?.value || "").trim();
      minhasSolicitacaoLastSearch = q;
      setStatus("Buscando minhas solicitaÃ§Ãµes");
      const res = await fetch("/api/solicitacoes/minhas?q=" + encodeURIComponent(q) + "&status=" + encodeURIComponent(status) + "&from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao buscar minhas solicitaÃ§Ãµes");
        renderMinhasSolicitacoesTable([]);
        return;
      }
      renderMinhasSolicitacoesTable(data.items || []);
      setStatus("Minhas solicitaÃ§Ãµes carregadas: " + String(data.count || 0));
    }

    async function saveSolicitacao(){
      const payload = gatherSolicitacaoForm();
      setStatus("Salvando solicitaÃ§Ã£o");
      const res = await fetch("/api/solicitacoes/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar solicitaÃ§Ã£o");
        return;
      }
      setStatus(data.message || "SolicitaÃ§Ã£o salva");
      await searchSolicitacoes();
      if (data.id) await openSolicitacaoEditModal(data.id);
    }

    async function deleteSolicitacao(id){
      if (!id) return;
      if (!confirm("Confirma excluir esta solicitacao")) return;
      setStatus("Excluindo solicitaÃ§Ã£o");
      const res = await fetch("/api/solicitacoes/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir solicitaÃ§Ã£o");
        return;
      }
      setStatus(data.message || "SolicitaÃ§Ã£o removida");
      await searchSolicitacoes();
    }

    function selectedAuthIDs(){
      const out = [];
      const checks = document.querySelectorAll(".auth-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllAuth(){
      const all = !!document.getElementById("auth_select_all").checked;
      const checks = document.querySelectorAll(".auth-row-check");
      for (const el of checks) el.checked = all;
    }

    function prepareAuthCreateForm(){
      clearAuthForm();
      document.getElementById("authModalTitle").textContent = "Novo UsuÃ¡rio";
    }

    function openAuthCreateModal(){
      prepareAuthCreateForm();
      document.getElementById("authEditModal").classList.remove("hidden");
    }

    async function openAuthEditModal(id){
      const row = authRowById(id);
      if (row) fillAuthForm(row);
      else clearAuthForm();

      setStatus("Carregando usuario");
      const res = await fetch("/api/auth/user/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar usuario");
        return;
      }
      fillAuthForm(data);
      document.getElementById("authModalTitle").textContent = "Editar UsuÃ¡rio";
      document.getElementById("authEditModal").classList.remove("hidden");
      setStatus("UsuÃ¡rio carregado");
    }

    function closeAuthEditModal(ev){
      if (ev && ev.target && ev.target.id !== "authEditModal") return;
      document.getElementById("authEditModal").classList.add("hidden");
    }

    function clearIDsGrupoForm(){
      for (const field of idsgFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
    }

    function fillIDsGrupoForm(data){
      if (!data) {
        clearIDsGrupoForm();
        return;
      }
      document.getElementById("idsg_id").value = data.id ?? "";
      document.getElementById("idsg_id_grupo").value = data.id_grupo ?? "";
      document.getElementById("idsg_produto").value = data.produto ?? "";
      document.getElementById("idsg_vencimento").value = data.due_day ?? "";
      document.getElementById("idsg_prazo").value = data.term_months ?? "";
      document.getElementById("idsg_tipo").value = data.group_kind ?? "";
      document.getElementById("idsg_grupo").value = data.group_code ?? "";
      document.getElementById("idsg_cota").value = data.quota ?? "";
      document.getElementById("idsg_r").value = data.r ?? "";
      document.getElementById("idsg_d").value = data.d ?? "";
      document.getElementById("idsg_parcelas_calc").value = data.parcelas_calc ?? "";
      document.getElementById("idsg_booked").value = data.booked ?? "";
      document.getElementById("idsg_created_at").value = data.created_at ?? "";
      document.getElementById("idsg_participantes").value = data.participants ?? "";
      document.getElementById("idsg_failed").value = data.failed ?? "";
    }

    function gatherIDsGrupoForm(){
      return {
        id: Number(document.getElementById("idsg_id").value || 0),
        id_grupo: Number(document.getElementById("idsg_id_grupo").value || 0),
        produto: document.getElementById("idsg_produto").value || "",
        due_day: Number(document.getElementById("idsg_vencimento").value || 0),
        term_months: Number(document.getElementById("idsg_prazo").value || 0),
        group_kind: document.getElementById("idsg_tipo").value || "",
        group_code: Number(document.getElementById("idsg_grupo").value || 0),
        quota: Number(document.getElementById("idsg_cota").value || 0),
        r: Number(document.getElementById("idsg_r").value || 0),
        d: Number(document.getElementById("idsg_d").value || 0),
        booked: Number(document.getElementById("idsg_booked").value || 0),
        created_at: document.getElementById("idsg_created_at").value || "",
        participants: Number(document.getElementById("idsg_participantes").value || 0),
        failed: Number(document.getElementById("idsg_failed").value || 0)
      };
    }

    function isEmptyIDsGrupoPayload(payload){
      if (!payload || payload.id > 0) return false;
      return !payload.id_grupo &&
        !String(payload.produto || "").trim() &&
        !payload.due_day &&
        !payload.term_months &&
        !String(payload.group_kind || "").trim() &&
        !payload.group_code &&
        !payload.quota &&
        !payload.r &&
        !payload.d &&
        !payload.booked &&
        !String(payload.created_at || "").trim() &&
        !payload.participants &&
        !payload.failed;
    }

    function renderIDsGruposTable(items, appendRows){
      const incoming = Array.isArray(items) ? items : [];
      if (appendRows) idsgRows = idsgRows.concat(incoming);
      else idsgRows = incoming;
      const body = document.getElementById("idsgTableBody");
      const selectAll = document.getElementById("idsg_select_all");
      if (selectAll && !appendRows) selectAll.checked = false;

      if (!idsgRows.length) {
        body.innerHTML = "<tr><td colspan=\"17\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      const rows = idsgRows.map((r) => {
        const id = Number(r.id || 0);
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"idsg-row-check\" value=\"" + String(id) + "\" /></td>" +
          "<td>" + escapeHtml(id) + "</td>" +
          "<td>" + escapeHtml(r.id_grupo || "") + "</td>" +
          "<td>" + escapeHtml(r.produto || "") + "</td>" +
          "<td>" + escapeHtml(r.due_day || "") + "</td>" +
          "<td>" + escapeHtml(r.term_months || "") + "</td>" +
          "<td>" + escapeHtml(r.group_kind || "") + "</td>" +
          "<td>" + escapeHtml(r.participants || "") + "</td>" +
          "<td>" + escapeHtml(r.group_code || "") + "</td>" +
          "<td>" + escapeHtml(r.quota || "") + "</td>" +
          "<td>" + escapeHtml(r.r || "") + "</td>" +
          "<td>" + escapeHtml(r.d || "") + "</td>" +
          "<td>" + escapeHtml(r.parcelas_calc || "") + "</td>" +
          "<td>" + escapeHtml(r.booked || "") + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(r.created_at || "")) + "</td>" +
          "<td>" + escapeHtml(r.failed || "") + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openIDsGrupoEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteIDsGrupo(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
      body.innerHTML = rows;
    }

    function selectedIDsGruposIDs(){
      const out = [];
      const checks = document.querySelectorAll(".idsg-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllIDsGrupos(){
      const all = !!document.getElementById("idsg_select_all").checked;
      const checks = document.querySelectorAll(".idsg-row-check");
      for (const el of checks) el.checked = all;
    }

    async function searchIDsGrupos(){
      const q = (document.getElementById("idsg_search").value || "").trim();
      const column = (document.getElementById("idsg_search_column").value || "").trim();
      idsgLastSearch = q;
      idsgOffset = 0;
      idsgTotal = 0;
      idsgHasMore = true;
      renderIDsGruposTable([], false);
      await loadMoreIDsGrupos();
    }

    async function loadMoreIDsGrupos(){
      if (idsgLoading || !idsgHasMore) return;
      idsgLoading = true;
      const q = (document.getElementById("idsg_search").value || "").trim();
      const column = (document.getElementById("idsg_search_column").value || "").trim();
      setStatus("Buscando ids grupos disponÃ­veis");
      const url = "/api/available_group_ids?q=" + encodeURIComponent(q) +
        "&column=" + encodeURIComponent(column) +
        "&offset=" + encodeURIComponent(String(idsgOffset)) +
        "&limit=" + encodeURIComponent(String(idsgLimit));
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        if (idsgOffset === 0) renderIDsGruposTable([], false);
        setStatus(data.message || "Erro na busca");
        idsgLoading = false;
        return;
      }
      const items = Array.isArray(data.items) ? data.items : [];
      idsgTotal = Number(data.total || 0);
      renderIDsGruposTable(items, idsgOffset > 0);
      idsgOffset += items.length;
      idsgHasMore = idsgOffset < idsgTotal;
      setStatus("Ids grupos carregados: " + String(idsgTotal || 0));
      idsgLoading = false;
    }

    function bindIDsGruposInfiniteScroll(){
      const wrap = document.getElementById("idsgTableWrap");
      if (!wrap || wrap.dataset.boundScroll === "1") return;
      wrap.dataset.boundScroll = "1";
      wrap.addEventListener("scroll", () => {
        if (idsgLoading || !idsgHasMore) return;
        const remain = wrap.scrollHeight - wrap.clientHeight - wrap.scrollTop;
        if (remain <= 180) {
          loadMoreIDsGrupos();
        }
      });
    }

    async function openIDsGrupoEditModal(id){
      setStatus("Carregando registro");
      const res = await fetch("/api/available_group_ids/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar registro");
        return;
      }
      fillIDsGrupoForm(data);
      const title = document.getElementById("idsgModalTitle");
      if (title) title.textContent = "Editar Ids Grupos DisponÃ­veis";
      document.getElementById("idsgEditModal").classList.remove("hidden");
      setStatus("Registro carregado");
    }

    function openIDsGrupoCreateModal(){
      clearIDsGrupoForm();
      const title = document.getElementById("idsgModalTitle");
      if (title) title.textContent = "Adicionar Ids Grupos DisponÃ­veis";
      document.getElementById("idsgEditModal").classList.remove("hidden");
      setStatus("Novo registro");
    }

    function closeIDsGrupoEditModal(ev){
      if (ev && ev.target && ev.target.id !== "idsgEditModal") return;
      document.getElementById("idsgEditModal").classList.add("hidden");
    }

    async function saveIDsGrupo(){
      const payload = gatherIDsGrupoForm();
      if (isEmptyIDsGrupoPayload(payload)) {
        setStatus("Preencha ao menos um campo para salvar");
        return;
      }
      setStatus("Salvando registro");
      const res = await fetch("/api/available_group_ids/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar");
        return;
      }
      closeIDsGrupoEditModal();
      setStatus(data.message || (payload.id ? "Registro atualizado" : "Registro criado"));
      await searchIDsGrupos();
    }

    async function deleteIDsGrupo(id){
      if (!confirm("Confirma excluir este registro")) return;
      setStatus("Excluindo registro");
      const res = await fetch("/api/available_group_ids/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir registro");
        return;
      }
      setStatus(data.message || "Registro removido");
      await searchIDsGrupos();
    }

    async function deleteSelectedIDsGrupos(){
      const ids = selectedIDsGruposIDs();
      if (!ids.length) {
        setStatus("Selecione ao menos um registro");
        return;
      }
      if (!confirm("Confirma excluir os registros selecionados")) return;
      setStatus("Excluindo registros selecionados");
      const res = await fetch("/api/available_group_ids/delete-batch", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ids})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir selecionados");
        return;
      }
      setStatus(data.message || "Registros removidos");
      await searchIDsGrupos();
    }

    function clearAssembleiaForm(){
      for (const field of assembleiaFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
    }

    function fillAssembleiaForm(data){
      if (!data) {
        clearAssembleiaForm();
        return;
      }
      let dataContemplacao = data.contemplation_date || "";
      if (dataContemplacao && String(dataContemplacao).length >= 10) {
        dataContemplacao = String(dataContemplacao).slice(0, 10);
      }
      document.getElementById("assembleia_id").value = data.id || "";
      document.getElementById("assembleia_cota_r_d").value = data.quota_rd || "";
      document.getElementById("assembleia_data_contemplacao").value = dataContemplacao;
      document.getElementById("assembleia_tipo_contemplacao").value = data.contemplation_type || "";
      document.getElementById("assembleia_data_desclassificao").value = data.disqualification_date || "";
      document.getElementById("assembleia_cliente").value = data.client_name || "";
      document.getElementById("assembleia_perc_lance").value = data.bid_percent || "";
      document.getElementById("assembleia_vendedor").value = data.seller_name || "";
      document.getElementById("assembleia_grupo").value = data.group_code || "";
      document.getElementById("assembleia_loteria_federal").value = data.federal_lottery || "";
      document.getElementById("assembleia_grupo_cota_r_d").value = data.group_quota_rd || "";
    }

    function gatherAssembleiaForm(){
      return {
        id: Number(document.getElementById("assembleia_id").value || 0),
        quota_rd: document.getElementById("assembleia_cota_r_d").value || "",
        contemplation_date: document.getElementById("assembleia_data_contemplacao").value || "",
        contemplation_type: document.getElementById("assembleia_tipo_contemplacao").value || "",
        disqualification_date: document.getElementById("assembleia_data_desclassificao").value || "",
        client_name: document.getElementById("assembleia_cliente").value || "",
        bid_percent: document.getElementById("assembleia_perc_lance").value || "",
        seller_name: document.getElementById("assembleia_vendedor").value || "",
        group_code: document.getElementById("assembleia_grupo").value || "",
        federal_lottery: document.getElementById("assembleia_loteria_federal").value || ""
      };
    }

    function isEmptyAssembleiaPayload(payload){
      if (!payload || payload.id > 0) return false;
      return !String(payload.quota_rd || "").trim() &&
        !String(payload.contemplation_date || "").trim() &&
        !String(payload.contemplation_type || "").trim() &&
        !String(payload.disqualification_date || "").trim() &&
        !String(payload.client_name || "").trim() &&
        !String(payload.bid_percent || "").trim() &&
        !String(payload.seller_name || "").trim() &&
        !String(payload.group_code || "").trim() &&
        !String(payload.federal_lottery || "").trim();
    }

    function renderAssembleiasTable(items){
      assembleiaRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("assembleiaTableBody");
      if (!assembleiaRows.length) {
        body.innerHTML = "<tr><td colspan=\"11\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = assembleiaRows.map((a) => {
        const id = Number(a.id || 0);
        return "<tr>" +
          "<td>" + escapeHtml(String(id)) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(a.contemplation_date || "")) + "</td>" +
          "<td>" + escapeHtml(String(a.federal_lottery || "")) + "</td>" +
          "<td>" + escapeHtml(String(a.group_code || "")) + "</td>" +
          "<td>" + escapeHtml(a.quota_rd || "") + "</td>" +
          "<td>" + escapeHtml(a.contemplation_type || "") + "</td>" +
          "<td>" + escapeHtml(a.client_name || "") + "</td>" +
          "<td>" + escapeHtml(formatPercent(a.bid_percent || "")) + "</td>" +
          "<td>" + escapeHtml(a.seller_name || "") + "</td>" +
          "<td>" + escapeHtml(a.group_quota_rd || "") + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openAssembleiaEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteAssembleia(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
    }

    async function searchAssembleias(){
      const q = (document.getElementById("assembleia_search").value || "").trim();
      assembleiaLastSearch = q;
      setStatus("Buscando assembleias");
      const res = await fetch("/api/assembleias?q=" + encodeURIComponent(q));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderAssembleiasTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderAssembleiasTable(data.items || []);
      setStatus("Assembleias carregadas: " + String(data.count || 0));
    }

    async function openAssembleiaEditModal(id){
      setStatus("Carregando assembleia");
      const res = await fetch("/api/assembleias/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar assembleia");
        return;
      }
      fillAssembleiaForm(data);
      const title = document.getElementById("assembleiaModalTitle");
      if (title) title.textContent = "Editar Assembleia";
      document.getElementById("assembleiaEditModal").classList.remove("hidden");
      setStatus("Assembleia carregada");
    }

    function openAssembleiaCreateModal(){
      clearAssembleiaForm();
      const title = document.getElementById("assembleiaModalTitle");
      if (title) title.textContent = "Adicionar Assembleia";
      document.getElementById("assembleiaEditModal").classList.remove("hidden");
      setStatus("Nova Assembleia");
    }

    function closeAssembleiaEditModal(ev){
      if (ev && ev.target && ev.target.id !== "assembleiaEditModal") return;
      document.getElementById("assembleiaEditModal").classList.add("hidden");
    }

    async function saveAssembleia(){
      const payload = gatherAssembleiaForm();
      if (isEmptyAssembleiaPayload(payload)) {
        setStatus("Preencha ao menos um campo para salvar");
        return;
      }
      setStatus("Salvando assembleia");
      const res = await fetch("/api/assembleias/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar assembleia");
        return;
      }
      closeAssembleiaEditModal();
      setStatus(data.message || "Assembleia salva");
      await searchAssembleias();
    }

    async function deleteAssembleia(id){
      if (!confirm("Confirma excluir este registro")) return;
      setStatus("Excluindo assembleia");
      const res = await fetch("/api/assembleias/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir assembleia");
        return;
      }
      setStatus(data.message || "Assembleia removida");
      await searchAssembleias();
    }

    function clearModeloForm(){
      for (const field of modeloFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const s = document.getElementById("modelo_status");
      if (s) s.value = "Ativo";
    }

    function normalizeActiveStatus(value){
      const v = String(value || "").trim().toLowerCase();
      if (!v) return "Ativo";
      if (v === "is_active" || v === "active" || v === "1" || v === "sim" || v === "ativo") return "Ativo";
      if (v === "inactive" || v === "0" || v === "nao" || v === "nÃ£o" || v === "inativo") return "Inativo";
      return value;
    }

    function fillModeloForm(data){
      if (!data) {
        clearModeloForm();
        return;
      }
      document.getElementById("modelo_id").value = data.id || "";
      document.getElementById("modelo_idmodelo").value = data.model_api_id || "";
      document.getElementById("modelo_nome").value = data.model_name || "";
      document.getElementById("modelo_status").value = normalizeActiveStatus(data.status || "Ativo");
    }

    function gatherModeloForm(){
      return {
        id: Number(document.getElementById("modelo_id").value || 0),
        model_api_id: Number(document.getElementById("modelo_idmodelo").value || 0),
        model_name: document.getElementById("modelo_nome").value || "",
        status: document.getElementById("modelo_status").value || ""
      };
    }

    function renderModelosTable(items){
      modeloRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("modeloTableBody");
      if (!modeloRows.length) {
        body.innerHTML = "<tr><td colspan=\"5\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = modeloRows.map((m) => {
        const id = Number(m.id || 0);
        return "<tr>" +
          "<td>" + escapeHtml(String(id)) + "</td>" +
          "<td>" + escapeHtml(String(m.model_api_id || "")) + "</td>" +
          "<td>" + escapeHtml(m.model_name || "") + "</td>" +
          "<td>" + escapeHtml(m.status || "") + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openModeloEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteModelo(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
    }

    async function searchModelos(){
      const q = (document.getElementById("modelo_search").value || "").trim();
      modeloLastSearch = q;
      setStatus("Buscando models");
      const res = await fetch("/api/models?q=" + encodeURIComponent(q));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderModelosTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderModelosTable(data.items || []);
      setStatus("models carregados: " + String(data.count || 0));
    }

    async function openModeloEditModal(id){
      setStatus("Carregando model_name");
      const res = await fetch("/api/models/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar model_name");
        return;
      }
      fillModeloForm(data);
      const title = document.getElementById("modeloModalTitle");
      if (title) title.textContent = "Editar Modelo";
      document.getElementById("modeloEditModal").classList.remove("hidden");
      setStatus("Modelo carregado");
    }

    function openModeloCreateModal(){
      clearModeloForm();
      const title = document.getElementById("modeloModalTitle");
      if (title) title.textContent = "Adicionar Modelo";
      document.getElementById("modeloEditModal").classList.remove("hidden");
      setStatus("Novo model_name");
    }

    function closeModeloEditModal(ev){
      if (ev && ev.target && ev.target.id !== "modeloEditModal") return;
      document.getElementById("modeloEditModal").classList.add("hidden");
    }

    async function saveModelo(){
      const payload = gatherModeloForm();
      if (!payload.model_api_id || !String(payload.model_name || "").trim()) {
        setStatus("Preencha ID Modelo e Modelo");
        return;
      }
      setStatus("Salvando model_name");
      const res = await fetch("/api/models/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar model_name");
        return;
      }
      closeModeloEditModal();
      setStatus(data.message || "Modelo salvo");
      await searchModelos();
    }

    async function deleteModelo(id){
      if (!confirm("Confirma excluir este model_name")) return;
      setStatus("Excluindo model_name");
      const res = await fetch("/api/models/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir model_name");
        return;
      }
      setStatus(data.message || "Modelo removido");
      await searchModelos();
    }

    function clearProdutoForm(){
      for (const field of produtoFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const s = document.getElementById("produto_status");
      if (s) s.value = "Ativo";
    }

    function fillProdutoForm(data){
      if (!data) {
        clearProdutoForm();
        return;
      }
      document.getElementById("produto_id").value = data.id || "";
      document.getElementById("produto_idproduto").value = data.product_api_id || "";
      document.getElementById("produto_nome").value = data.produto || data.product_name || "";
      document.getElementById("produto_status").value = normalizeActiveStatus(data.status || "Ativo");
    }

    function gatherProdutoForm(){
      return {
        id: Number(document.getElementById("produto_id").value || 0),
        product_api_id: Number(document.getElementById("produto_idproduto").value || 0),
        produto: document.getElementById("produto_nome").value || "",
        status: document.getElementById("produto_status").value || ""
      };
    }

    function renderProdutosTable(items){
      produtoRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("produtoTableBody");
      if (!produtoRows.length) {
        body.innerHTML = "<tr><td colspan=\"5\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = produtoRows.map((p) => {
        const id = Number(p.id || 0);
        return "<tr>" +
          "<td>" + escapeHtml(String(id)) + "</td>" +
          "<td>" + escapeHtml(String(p.product_api_id || p.IDProduto || "")) + "</td>" +
          "<td>" + escapeHtml(p.produto || p.product_name || p.Produto || "") + "</td>" +
          "<td>" + escapeHtml(p.status || "") + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openProdutoEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteProduto(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
    }

    async function searchProdutos(){
      const q = (document.getElementById("produto_search").value || "").trim();
      produtoLastSearch = q;
      setStatus("Buscando produtos");
      const res = await fetch("/api/produtos?q=" + encodeURIComponent(q));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderProdutosTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderProdutosTable(data.items || []);
      setStatus("Produtos carregados: " + String(data.count || 0));
    }

    async function openProdutoEditModal(id){
      setStatus("Carregando produto");
      const res = await fetch("/api/produtos/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar produto");
        return;
      }
      fillProdutoForm(data);
      const title = document.getElementById("produtoModalTitle");
      if (title) title.textContent = "Editar Produto";
      document.getElementById("produtoEditModal").classList.remove("hidden");
      setStatus("Produto carregado");
    }

    function openProdutoCreateModal(){
      clearProdutoForm();
      const title = document.getElementById("produtoModalTitle");
      if (title) title.textContent = "Adicionar Produto";
      document.getElementById("produtoEditModal").classList.remove("hidden");
      setStatus("Novo produto");
    }

    function closeProdutoEditModal(ev){
      if (ev && ev.target && ev.target.id !== "produtoEditModal") return;
      document.getElementById("produtoEditModal").classList.add("hidden");
    }

    async function saveProduto(){
      const payload = gatherProdutoForm();
      if (!payload.product_api_id || !String(payload.produto || "").trim()) {
        setStatus("Preencha ID Produto e Produto");
        return;
      }
      setStatus("Salvando produto");
      const res = await fetch("/api/produtos/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar produto");
        return;
      }
      closeProdutoEditModal();
      setStatus(data.message || "Produto salvo");
      await searchProdutos();
    }

    async function deleteProduto(id){
      if (!confirm("Confirma excluir este produto")) return;
      setStatus("Excluindo produto");
      const res = await fetch("/api/produtos/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir produto");
        return;
      }
      setStatus(data.message || "Produto removido");
      await searchProdutos();
    }

    function clearGrupoAtivoForm(){
      for (const field of gruposAtivosFields) {
        const el = document.getElementById(field);
        if (el) el.value = "";
      }
      const s = document.getElementById("ga_status");
      if (s) s.value = "is_active";
    }

    function fillGrupoAtivoForm(data){
      if (!data) {
        clearGrupoAtivoForm();
        return;
      }
      document.getElementById("ga_id").value = data.id || "";
      document.getElementById("ga_grupo").value = data.group_code || "";
      document.getElementById("ga_vencimento").value = data.due_day || "";
      document.getElementById("ga_qtd_participantes").value = data.participants_count || "";
      document.getElementById("ga_data_assembleia_inaugural").value = data.first_assembly_date || "";
      document.getElementById("ga_plano").value = data.plan || "";
      document.getElementById("ga_prazo").value = data.term_months || "";
      document.getElementById("ga_tipo_grupo").value = data.group_type || "";
      document.getElementById("ga_modelos").value = data.modelos || data.models || "";
      document.getElementById("ga_status").value = normalizeActiveStatus(data.status || "Ativo") === "Inativo" ? "inactive" : "is_active";
      document.getElementById("ga_created_at").value = data.created_at || "";
      document.getElementById("ga_updated_at").value = data.updated_at || "";
    }

    function gatherGrupoAtivoForm(){
      return {
        id: Number(document.getElementById("ga_id").value || 0),
        group_code: Number(document.getElementById("ga_grupo").value || 0),
        due_day: Number(document.getElementById("ga_vencimento").value || 0),
        participants_count: Number(document.getElementById("ga_qtd_participantes").value || 0),
        first_assembly_date: document.getElementById("ga_data_assembleia_inaugural").value || "",
        plan: document.getElementById("ga_plano").value || "",
        term_months: Number(document.getElementById("ga_prazo").value || 0),
        group_type: document.getElementById("ga_tipo_grupo").value || "",
        Modelos: document.getElementById("ga_modelos").value || "",
        status: document.getElementById("ga_status").value || "is_active",
        created_at: document.getElementById("ga_created_at").value || "",
        updated_at: document.getElementById("ga_updated_at").value || "",
      };
    }

    function renderGruposAtivosTable(items){
      gruposAtivosRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("gaTableBody");
      const selectAll = document.getElementById("ga_select_all");
      if (selectAll) selectAll.checked = false;
      if (!gruposAtivosRows.length) {
        body.innerHTML = "<tr><td colspan=\"13\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = gruposAtivosRows.map((g) => {
        const id = Number(g.id || 0);
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"ga-row-check\" value=\"" + String(id) + "\" /></td>" +
          "<td>" + escapeHtml(String(id)) + "</td>" +
          "<td>" + escapeHtml(String(g.group_code || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.due_day || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.participants_count || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.first_assembly_date || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.plan || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.term_months || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.parcelas_calculadas || "")) + "</td>" +
          "<td>" + escapeHtml(String(g.group_type || "")) + "</td>" +
          "<td title=\"" + escapeHtml(String(g.modelos || g.models || "")) + "\"><span class=\"ga-cell-ellipsis\">" + escapeHtml(String(g.modelos || g.models || "")) + "</span></td>" +
          "<td>" + escapeHtml(normalizeActiveStatus(String(g.status || ""))) + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Editar\" aria-label=\"Editar\" onclick=\"openGrupoAtivoEditModal(" + String(id) + ")\">" + authActionIcon("edit") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteGrupoAtivo(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
    }

    async function searchGruposAtivos(){
      const q = (document.getElementById("ga_search").value || "").trim();
      const column = (document.getElementById("ga_search_column").value || "").trim();
      gruposAtivosLastSearch = q;
      setStatus("Buscando grupos ativos");
      const res = await fetch("/api/active_groups?q=" + encodeURIComponent(q) + "&column=" + encodeURIComponent(column));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderGruposAtivosTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderGruposAtivosTable(data.items || []);
      setStatus("Grupos ativos carregados: " + String(data.count || 0));
    }

    async function openGrupoAtivoEditModal(id){
      setStatus("Carregando grupo ativo");
      const res = await fetch("/api/active_groups/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao carregar grupo ativo");
        return;
      }
      fillGrupoAtivoForm(data);
      const title = document.getElementById("gaModalTitle");
      if (title) title.textContent = "Editar Grupo Ativo";
      document.getElementById("gaEditModal").classList.remove("hidden");
      setStatus("Grupo ativo carregado");
    }

    function openGrupoAtivoCreateModal(){
      clearGrupoAtivoForm();
      const title = document.getElementById("gaModalTitle");
      if (title) title.textContent = "Adicionar Grupo Ativo";
      document.getElementById("gaEditModal").classList.remove("hidden");
      setStatus("Novo grupo ativo");
    }

    function closeGrupoAtivoEditModal(ev){
      if (ev && ev.target && ev.target.id !== "gaEditModal") return;
      document.getElementById("gaEditModal").classList.add("hidden");
    }

    async function saveGrupoAtivo(){
      const payload = gatherGrupoAtivoForm();
      if (!payload.group_code || !payload.due_day) {
        setStatus("Preencha Grupo e Vencimento");
        return;
      }
      setStatus("Salvando grupo ativo");
      const res = await fetch("/api/active_groups/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar grupo ativo");
        return;
      }
      closeGrupoAtivoEditModal();
      setStatus(data.message || "Grupo ativo salvo");
      await searchGruposAtivos();
    }

    async function deleteGrupoAtivo(id){
      if (!confirm("Confirma excluir este grupo ativo")) return;
      setStatus("Excluindo grupo ativo");
      const res = await fetch("/api/active_groups/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir grupo ativo");
        return;
      }
      setStatus(data.message || "Grupo ativo removido");
      await searchGruposAtivos();
    }

    function selectedGruposAtivosIDs(){
      const out = [];
      const checks = document.querySelectorAll(".ga-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllGruposAtivos(){
      const all = !!document.getElementById("ga_select_all").checked;
      const checks = document.querySelectorAll(".ga-row-check");
      for (const el of checks) el.checked = all;
    }

    async function deleteSelectedGruposAtivos(){
      const ids = selectedGruposAtivosIDs();
      if (!ids.length) {
        setStatus("Selecione ao menos um registro");
        return;
      }
      if (!confirm("Confirma excluir os registros selecionados")) return;
      setStatus("Excluindo grupos ativos selecionados");
      const res = await fetch("/api/active_groups/delete-batch", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ids})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir registros");
        return;
      }
      setStatus(data.message || "Registros removidos");
      await searchGruposAtivos();
    }

    async function searchAuthUsers(){
      const q = (document.getElementById("auth_search").value || "").trim();
      authLastSearch = q;
      setStatus("Buscando usuarios");
      const res = await fetch("/api/auth/users?q=" + encodeURIComponent(q));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderAuthTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderAuthTable(data.items || []);
      setStatus("UsuÃ¡rios carregados: " + String(data.count || 0));
    }

    async function saveAuthUser(){
      const payload = gatherAuthForm();
      if (!payload.cpf || !payload.company_code || !payload.account_user || (!payload.id && !payload.account_password)) {
        setStatus("Preencha CPF, Cod. Empresa, Cod. Usuario e senha (obrigatoria no cadastro novo)");
        return;
      }
      setStatus("Salvando usuario");
      const res = await fetch("/api/auth/user/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao salvar usuario");
        return;
      }
      if (data.user) {
        fillAuthForm(data.user);
        document.getElementById("authModalTitle").textContent = "Editar UsuÃ¡rio";
      } else if (data.id) {
        const getRes = await fetch("/api/auth/user/get?id=" + encodeURIComponent(String(data.id)));
        const getData = await getRes.json();
        if (getRes.ok && getData.ok !== false) {
          fillAuthForm(getData);
          document.getElementById("authModalTitle").textContent = "Editar UsuÃ¡rio";
        }
      }
      document.getElementById("authEditModal").classList.remove("hidden");
      setStatus(data.message || "UsuÃ¡rio salvo");
      await searchAuthUsers();
    }

    async function deleteAuthUser(id){
      if (!confirm("Confirma excluir este usuario")) return;
      setStatus("Excluindo usuario");
      const res = await fetch("/api/auth/user/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir usuario");
        return;
      }
      setStatus(data.message || "UsuÃ¡rio removido");
      await searchAuthUsers();
    }

    async function authenticateAuthUser(id){
      setStatus("Autenticando usuario");
      const res = await fetch("/api/auth/user/login", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha na autenticaÃ§Ã£o");
        return;
      }
      setStatus(data.message || "UsuÃ¡rio autenticado");
      await searchAuthUsers();
    }

    async function authenticateSelectedUsers(){
      const ids = selectedAuthIDs();
      if (!ids.length) {
        setStatus("Selecione ao menos um usuario");
        return;
      }
      setStatus("Autenticando selecionados");
      const res = await fetch("/api/auth/users/login", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ids})
      });
      const data = await res.json();
      setStatus(data.message || "AutenticaÃ§Ã£o concluÃ­da");
      await searchAuthUsers();
    }

    async function reportUnauthenticatedUsers(){
      if (!Array.isArray(authRows) || !authRows.length) {
        setStatus("Nenhum usuario carregado para analise");
        return;
      }
      const missing = authRows.filter((u) => !String(u.token || "").trim());
      if (!missing.length) {
        const msgOk = "Todos os usuarios listados estao autenticados (token preenchido).";
        setStatus(msgOk);
        return;
      }

      const hour = new Date().getHours();
      let saudacao = "Bom dia,";
      if (hour >= 12 && hour < 18) saudacao = "Boa tarde,";
      if (hour >= 18 || hour < 6) saudacao = "Boa noite,";

      const lines = [];
      lines.push(saudacao);
      lines.push("");
      lines.push("Solicitamos a atualização dos usuários listados abaixo. Por favor, desconsiderar aqueles que estiverem em período de férias.");
      lines.push("");
      for (const u of missing) {
        const company = String(u.company_code || "").replace(/\D+/g, "").trim();
        const user = String(u.account_user || "").trim();
        lines.push(company + " " + user);
      }
      const msg = lines.join("\n");

      try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(msg);
          setStatus("Lista de não autenticados copiada: " + String(missing.length) + " usuário(s) | " + msg);
          return;
        }
      } catch (_) {}
      setStatus("Lista de não autenticados: " + String(missing.length) + " usuário(s) | " + msg);
    }

    function renderReservedTable(items){
      reservedRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("reservedTableBody");
      const selectAll = document.getElementById("reserved_select_all");
      if (selectAll) selectAll.checked = false;

      if (!reservedRows.length) {
        body.innerHTML = "<tr><td colspan=\"9\">Nenhum registro encontrado.</td></tr>";
        return;
      }

      const rows = reservedRows.map((r) => {
        const id = Number(r.id || 0);
        const group_code = String(r.cod_grupo || r.group_code || "").trim();
        const quota_rd = String(r.cota_rd || "").trim();
        let grupoCotaRD = "";
        if (group_code && quota_rd) grupoCotaRD = group_code + "-" + quota_rd;
        else if (group_code) grupoCotaRD = group_code;
        else if (quota_rd) grupoCotaRD = quota_rd;
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"reserved-row-check\" value=\"" + String(id) + "\" /></td>" +
          "<td>" + escapeHtml(id) + "</td>" +
          "<td>" + escapeHtml(r.usuario_reserva || r.UsuarioReserva || "") + "</td>" +
          "<td>" + escapeHtml(r.num_documento_pessoa || r.NumDocumentoPessoa || "") + "</td>" +
          "<td>" + escapeHtml(r.cod_modelo || r.CodModelo || "") + "</td>" +
          "<td>" + escapeHtml(r.id_cota_reposicao || r.IDCotaReposicao || "") + "</td>" +
          "<td>" + escapeHtml(grupoCotaRD) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(r.created_at || "")) + "</td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteReservedCota(" + String(id) + ")\">" + authActionIcon("delete") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
      body.innerHTML = rows;
    }

    function selectedReservedIDs(){
      const out = [];
      const checks = document.querySelectorAll(".reserved-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllReserved(){
      const all = !!document.getElementById("reserved_select_all").checked;
      const checks = document.querySelectorAll(".reserved-row-check");
      for (const el of checks) el.checked = all;
    }

    async function searchReservedCotas(){
      const q = (document.getElementById("reserved_search").value || "").trim();
      const from = (document.getElementById("reserved_from").value || "").trim();
      const to = (document.getElementById("reserved_to").value || "").trim();
      reservedLastSearch = q;
      setStatus("Buscando cotas reservadas");
      const res = await fetch("/api/cotasreservadas?q=" + encodeURIComponent(q) + "&from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderReservedTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderReservedTable(data.items || []);
      setStatus("Cotas reservadas carregadas: " + String(data.count || 0));
    }

    function renderManualNotificationsTable(items){
      manualNotificationRows = Array.isArray(items) ? items : [];
      const body = document.getElementById("msgTableBody");
      const selectAll = document.getElementById("msg_select_all");
      if (selectAll) selectAll.checked = false;
      if (!body) return;
      if (!manualNotificationRows.length) {
        body.innerHTML = "<tr><td colspan=\"13\">Nenhum registro encontrado.</td></tr>";
        return;
      }
      body.innerHTML = manualNotificationRows.map((m) => {
        const id = Number(m.id || 0);
        return "<tr>" +
          "<td><input type=\"checkbox\" class=\"msg-row-check\" value=\"" + String(id) + "\" /></td>" +
          "<td>" + escapeHtml(String(id)) + "</td>" +
          "<td>" + escapeHtml(String(m.solicitacao_id || "")) + "</td>" +
          "<td>" + escapeHtml(m.seller_name || "") + "</td>" +
          "<td>" + escapeHtml(m.branch || "") + "</td>" +
          "<td>" + escapeHtml(m.cpf || "") + "</td>" +
          "<td>" + escapeHtml(m.channel || "") + "</td>" +
          "<td>" + escapeHtml(m.status || "") + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(m.created_at || "")) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(m.copied_at || "")) + "</td>" +
          "<td>" + escapeHtml(formatDateTimeBR(m.sent_at || "")) + "</td>" +
          "<td title=\"" + escapeHtml(m.message || "") + "\"><span class=\"sol-cell-ellipsis\">" + escapeHtml(m.message || "") + "</span></td>" +
          "<td><div class=\"auth-actions\">" +
            "<button type=\"button\" class=\"auth-action-btn\" title=\"Copiar\" aria-label=\"Copiar\" onclick=\"copyManualNotificationsByIDs([" + String(id) + "])\">" + authActionIcon("logs") + "</button>" +
            "<button type=\"button\" class=\"auth-action-btn auth-action-success\" title=\"Marcar enviada\" aria-label=\"Marcar enviada\" onclick=\"markManualNotificationsSentByIDs([" + String(id) + "])\">" + authActionIcon("reserve") + "</button>" +
          "</div></td>" +
        "</tr>";
      }).join("");
    }

    function selectedManualNotificationIDs(){
      const out = [];
      const checks = document.querySelectorAll(".msg-row-check:checked");
      for (const el of checks) {
        const id = Number(el.value || 0);
        if (id > 0) out.push(id);
      }
      return out;
    }

    function toggleSelectAllManualNotifications(){
      const all = !!document.getElementById("msg_select_all").checked;
      const checks = document.querySelectorAll(".msg-row-check");
      for (const el of checks) el.checked = all;
    }

    async function searchManualNotifications(){
      const q = (document.getElementById("msg_search").value || "").trim();
      const status = (document.getElementById("msg_status").value || "pendente").trim();
      const from = (document.getElementById("msg_from").value || "").trim();
      const to = (document.getElementById("msg_to").value || "").trim();
      manualNotificationLastSearch = q;
      setStatus("Buscando mensagens");
      const res = await fetch("/api/notificacoes/manual?q=" + encodeURIComponent(q) + "&status=" + encodeURIComponent(status) + "&from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        renderManualNotificationsTable([]);
        setStatus(data.message || "Erro na busca");
        return;
      }
      renderManualNotificationsTable(data.items || []);
      setStatus("Mensagens carregadas: " + String(data.count || 0));
    }

    async function setManualNotificationsStatus(ids, status){
      if (!Array.isArray(ids) || !ids.length) {
        setStatus("Selecione ao menos uma message");
        return false;
      }
      const res = await fetch("/api/notificacoes/manual/status", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ids, status})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Falha ao atualizar mensagens");
        return false;
      }
      return true;
    }

    async function copyManualNotificationsByIDs(ids){
      const wanted = new Set((ids || []).map((v) => Number(v || 0)).filter((v) => v > 0));
      const selected = manualNotificationRows.filter((m) => wanted.has(Number(m.id || 0)));
      if (!selected.length) {
        setStatus("Nenhuma message selecionada");
        return;
      }
      const text = selected.map((m) => String(m.message || "").trim()).filter((s) => s !== "").join("\n\n");
      if (!text) {
        setStatus("Nada para copiar");
        return;
      }
      try {
        await navigator.clipboard.writeText(text);
      } catch (_) {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
      }
      await setManualNotificationsStatus(ids, "copiada");
      await searchManualNotifications();
      setStatus("Mensagens copiadas");
    }

    async function copySelectedManualNotifications(){
      const ids = selectedManualNotificationIDs();
      await copyManualNotificationsByIDs(ids);
    }

    async function markManualNotificationsSentByIDs(ids){
      if (!Array.isArray(ids) || !ids.length) {
        setStatus("Selecione ao menos uma message");
        return;
      }
      const ok = await setManualNotificationsStatus(ids, "enviada_manual");
      if (!ok) return;
      await searchManualNotifications();
      setStatus("Mensagens marcadas como enviadas");
    }

    async function markSelectedManualNotificationsSent(){
      const ids = selectedManualNotificationIDs();
      await markManualNotificationsSentByIDs(ids);
    }

    async function deleteReservedCota(id){
      if (!confirm("Confirma excluir este registro")) return;
      setStatus("Excluindo registro");
      const res = await fetch("/api/cotasreservadas/delete", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({id})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir registro");
        return;
      }
      setStatus(data.message || "Registro removido");
      await searchReservedCotas();
    }

    async function deleteSelectedReservedCotas(){
      const ids = selectedReservedIDs();
      if (!ids.length) {
        setStatus("Selecione ao menos um registro");
        return;
      }
      if (!confirm("Confirma excluir os registros selecionados")) return;
      setStatus("Excluindo registros selecionados");
      const res = await fetch("/api/cotasreservadas/delete-batch", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ids})
      });
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao excluir selecionados");
        return;
      }
      setStatus(data.message || "Registros removidos");
      await searchReservedCotas();
    }

    async function runDatabaseAction(url, runningText, fallbackOkText){
      setStatus(runningText);
      const res = await fetch(url, {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(gather())
      });
      const data = await res.json();
      setStatus(data.message || (data.ok ? fallbackOkText : "Falha na operaÃ§Ã£o de banco"));
    }

    async function createDatabaseTables(){
      await runDatabaseAction("/api/db/tables/create", "Criando/atualizando tabelas", "Tabelas criadas/atualizadas");
    }

    async function clearDatabaseTables(){
      if (!confirm("Confirma excluir todos os dados das tabelas legadas")) return;
      await runDatabaseAction("/api/db/tables/clear", "Excluindo dados das tabelas", "Dados das tabelas excluÃ­dos");
    }

    async function dropDatabaseTables(){
      if (!confirm("Confirma excluir as tabelas legadas do banco")) return;
      await runDatabaseAction("/api/db/tables/drop", "Excluindo tabelas do banco", "Tabelas excluÃ­das");
    }

    async function downloadDatabaseBackup() {
      try {
        setStatus("Gerando arquivo de backup...");
        const res = await fetch("/api/db/backup", { method: "GET" });
        if (!res.ok) {
          let msg = "Erro ao gerar backup";
          try {
            const data = await res.json();
            if (data && data.message) msg = data.message;
          } catch (_) {}
          setStatus(msg);
          return;
        }

        const blob = await res.blob();
        const contentDisposition = res.headers.get("content-disposition") || "";
        let fileName = "honda_go_backup.sql";
        const m = contentDisposition.match(/filename=\"?([^\";]+)\"?/i);
        if (m && m[1]) fileName = m[1];

        const link = document.createElement("a");
        const url = URL.createObjectURL(blob);
        link.href = url;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
        setStatus("Backup gerado: " + fileName);
      } catch (err) {
        setStatus("Erro ao baixar backup: " + err.message);
      }
    }

    async function uploadDatabaseRestore(input) {
      const file = input.files[0];
      if (!file) return;

      if (!confirm("AVISO: Restaurar o banco ira EXCLUIR todos os dados atuais e substitui-los pelos dados do arquivo. Deseja continuar?")) {
        input.value = "";
        return;
      }

      try {
        setStatus("Enviando arquivo e restaurando banco...");
        const formData = new FormData();
        formData.append("file", file);

        const res = await fetch("/api/db/restore", {
          method: "POST",
          body: formData
        });

        const data = await res.json();
        if (!res.ok || data.ok === false) {
          setStatus(data.message || "Erro ao restaurar banco");
        } else {
          setStatus(data.message || "Banco restaurado com sucesso");
          alert("Banco de dados restaurado com sucesso!");
        }
      } catch (err) {
        setStatus("Erro na restauraÃ§Ã£o: " + err.message);
      } finally {
        input.value = "";
      }
    }

    async function refreshDBRestoreCapabilities() {
      const backupBtn = document.getElementById("db_backup_btn");
      const restoreBtn = document.getElementById("db_restore_btn");
      const restoreInput = document.getElementById("db_restore_file");
      if (!restoreBtn || !restoreInput || !backupBtn) return;

      try {
        const res = await fetch("/api/db/restore-capabilities", { method: "GET" });
        const data = await res.json();
        const restoreAvailable = !!(res.ok && data && data.ok !== false && data.restore_available);
        const backupAvailable = !!(res.ok && data && data.ok !== false && data.backup_available);

        if (!backupAvailable) {
          backupBtn.disabled = true;
          backupBtn.title = (data && data.backup_reason) ? data.backup_reason : "Backup indisponivel neste ambiente.";
        } else {
          backupBtn.disabled = false;
          backupBtn.title = "";
        }

        if (!restoreAvailable) {
          restoreBtn.disabled = true;
          restoreBtn.title = (data && data.restore_reason) ? data.restore_reason : "Restore indisponivel neste ambiente.";
        } else {
          restoreBtn.disabled = false;
          restoreBtn.title = "";
        }
      } catch (_) {
        backupBtn.disabled = true;
        backupBtn.title = "Backup indisponivel neste ambiente.";
        restoreBtn.disabled = true;
        restoreBtn.title = "Restore indisponivel neste ambiente.";
      }
    }


    async function authenticateSelectedUser(){
      const id = Number(document.getElementById("auth_id").value || 0);
      if (!id) {
        setStatus("UsuÃ¡rio sem ID para autenticar");
        return;
      }
      await authenticateAuthUser(id);
      const res = await fetch("/api/auth/user/get?id=" + encodeURIComponent(String(id)));
      const data = await res.json();
      if (res.ok && data.ok !== false) fillAuthForm(data);
    }

    async function loadConfig(){
      setStatus("Carregando");
      const path = document.getElementById("config_path").value || "config.ini";
      const res = await fetch("/api/config/load?path=" + encodeURIComponent(path));
      const data = await res.json();
      if (!res.ok || data.ok === false) {
        setStatus(data.message || "Erro ao carregar");
        return;
      }
      fill(data);
      await refreshDBRestoreCapabilities();
      setStatus("Config carregada");
    }

    async function saveConfig(){
      setStatus("Salvando");
      const res = await fetch("/api/config/save", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(gather())
      });
      const data = await res.json();
      setStatus(data.message || (data.ok ? "Config salva" : "Erro ao salvar"));
    }

    async function runEngine(){
      setStatus("Iniciando");
      const res = await fetch("/api/run", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(gather())
      });
      const data = await res.json();
      setStatus(data.message || (data.ok ? "Executando" : "Erro ao iniciar"));
    }

    async function authenticateUsers(){
      setStatus("Autenticando");
      const res = await fetch("/api/auth", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify(gather())
      });
      const data = await res.json();
      setStatus(data.message || (data.ok ? "Autenticando" : "Erro na autenticaÃ§Ã£o"));
    }

    async function stopEngine(){
      const res = await fetch("/api/stop", {method:"POST"});
      const data = await res.json();
      setStatus(data.message || "Parada solicitada");
    }

    async function clearLogs(){
      const res = await fetch("/api/logs/clear", {method:"POST"});
      const data = await res.json();
      if (res.ok && data.ok !== false) {
        document.getElementById("logs").textContent = "";
      }
      setStatus(data.message || "Log limpo");
    }

    function shouldStickToBottom(el){
      if (!el) return true;
      const distance = el.scrollHeight - el.clientHeight - el.scrollTop;
      return distance <= 24;
    }

    async function refreshLogs(){
      if (!isAuthenticated) return;
      const res = await fetch("/api/logs");
      const data = await res.json();
      const logs = document.getElementById("logs");
      const stickToBottom = shouldStickToBottom(logs);
      const prevTop = logs.scrollTop;
      logs.textContent = data.logs || "";
      if (stickToBottom) logs.scrollTop = logs.scrollHeight;
      else logs.scrollTop = prevTop;
    }

    async function refreshStatus(){
      if (!isAuthenticated) return;
      const res = await fetch("/api/status");
      const data = await res.json();
      if (data.running) setStatus("Executando");
    }

    function formatBytes(bytes){
      const n = Number(bytes || 0);
      if (n < 1024) return n + " B";
      const kb = n / 1024;
      if (kb < 1024) return kb.toFixed(1) + " KB";
      const mb = kb / 1024;
      return mb.toFixed(2) + " MB";
    }

    function formatDuration(ms){
      const total = Math.max(0, Math.floor((ms || 0) / 1000));
      const h = Math.floor(total / 3600);
      const m = Math.floor((total % 3600) / 60);
      const s = total % 60;
      if (h > 0) return String(h).padStart(2, "0") + ":" + String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
      return String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
    }

    function formatDateTimeBR(value){
      const s = String(value || "").trim();
      if (!s) return "";
      const m = s.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/);
      if (m) return m[3] + "/" + m[2] + "/" + m[1] + " " + m[4] + ":" + m[5];
      const d = new Date(s);
      if (!Number.isNaN(d.getTime())) {
        const dd = String(d.getDate()).padStart(2, "0");
        const mm = String(d.getMonth() + 1).padStart(2, "0");
        const yy = d.getFullYear();
        const hh = String(d.getHours()).padStart(2, "0");
        const mi = String(d.getMinutes()).padStart(2, "0");
        return dd + "/" + mm + "/" + yy + " " + hh + ":" + mi;
      }
      return s;
    }

    function formatDateTimeBRSeconds(value){
      const s = String(value || "").trim();
      if (!s) return "";
      const m = s.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})/);
      if (m) return m[3] + "/" + m[2] + "/" + m[1] + " " + m[4] + ":" + m[5] + ":" + m[6];
      const d = new Date(s);
      if (!Number.isNaN(d.getTime())) {
        const dd = String(d.getDate()).padStart(2, "0");
        const mm = String(d.getMonth() + 1).padStart(2, "0");
        const yy = d.getFullYear();
        const hh = String(d.getHours()).padStart(2, "0");
        const mi = String(d.getMinutes()).padStart(2, "0");
        const ss = String(d.getSeconds()).padStart(2, "0");
        return dd + "/" + mm + "/" + yy + " " + hh + ":" + mi + ":" + ss;
      }
      return s;
    }

    function formatPercent(value){
      const s = String(value || "").trim();
      if (!s) return "";
      const clean = s.endsWith("%") ? s.slice(0, -1).trim() : s;
      const n = Number(clean.replace(",", "."));
      if (!Number.isFinite(n)) return s;
      return n.toLocaleString("pt-BR", { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + "%";
    }

    function statusCodeFromLine(line){
      let match = line.match(/\bstatus=(\d{3})\b/i);
      if (!match) match = line.match(/\bResponse\s+(\d{3})\b/i);
      if (!match) match = line.match(/\bError\s+(\d{3})\b/i);
      if (!match) return 0;
      const code = Number(match[1]);
      return Number.isFinite(code) ? code : 0;
    }

    function isInterestingLine(line){
      const l = String(line || "").toLowerCase();
      return l.includes("[go][metrics]") ||
        l.includes("response ") ||
        l.includes("error ") ||
        l.includes("[go] finished in") ||
        l.includes("[go] run engine:") ||
        l.includes("[go][dry-run]");
    }

    function closeMonitoringModal(ev){
      if (ev && ev.target && ev.target.id !== "monitorModal") return;
      document.getElementById("monitorModal").classList.add("hidden");
      if (monitorPollTimer) {
        clearInterval(monitorPollTimer);
        monitorPollTimer = null;
      }
    }

    async function openMonitoringModal(){
      document.getElementById("monitorModal").classList.remove("hidden");
      await refreshMonitoring();
      if (!monitorPollTimer) {
        monitorPollTimer = setInterval(refreshMonitoring, 700);
      }
    }

    async function refreshMonitoring(){
      if (!isAuthenticated) return;
      try {
        const [statusRes, logsRes] = await Promise.all([
          fetch("/api/status"),
          fetch("/api/logs"),
        ]);

        const statusData = await statusRes.json();
        const logsData = await logsRes.json();
        const allLogs = String(logsData.logs || "");
        const lines = allLogs.split(/\r\n/).map((x) => x.trim()).filter((x) => x.length > 0);

        const running = !!statusData.running;
        const startedAtMs = statusData.started_at ? Date.parse(statusData.started_at) : NaN;
        const finishedAtMs = statusData.finished_at ? Date.parse(statusData.finished_at) : NaN;

        let elapsedMs = 0;
        if (Number.isFinite(startedAtMs)) {
          if (running) elapsedMs = Date.now() - startedAtMs;
          else if (Number.isFinite(finishedAtMs)) elapsedMs = Math.max(0, finishedAtMs - startedAtMs);
        }

        let http2xx = 0;
        let http4xx = 0;
        let http5xx = 0;
        let responses = 0;
        for (const line of lines) {
          const code = statusCodeFromLine(line);
          if (!code) continue;
          responses++;
          if (code >= 200 && code < 300) http2xx++;
          else if (code >= 400 && code < 500) http4xx++;
          else if (code >= 500 && code < 600) http5xx++;
        }

        const durationSec = elapsedMs > 0 ? (elapsedMs / 1000) : 0;
        const linesRate = durationSec > 0 ? (lines.length / durationSec) : 0;
        const recent = lines.filter(isInterestingLine);
        const recentLines = (recent.length ? recent : lines).slice(-24);

        document.getElementById("monStatus").textContent = running ? "Executando" : "Parado";
        document.getElementById("monElapsed").textContent = elapsedMs > 0 ? ("Tempo: " + formatDuration(elapsedMs)) : "Tempo: --";
        document.getElementById("monLines").textContent = String(lines.length);
        document.getElementById("monRate").textContent = linesRate.toFixed(2) + " linhas/s";
        document.getElementById("monHttp").textContent = String(http2xx) + " / " + String(http4xx) + " / " + String(http5xx);
        document.getElementById("monHttpDetail").textContent = String(responses) + " respostas";
        document.getElementById("monUpdated").textContent = new Date().toLocaleTimeString();
        document.getElementById("monWindow").textContent = running ? "atualizaÃ§Ã£o ativa" : "sem execuÃ§Ã£o ativa";

        const monInfo = document.getElementById("monInfo");
        const monRecent = document.getElementById("monRecent");
        const stickToBottom = shouldStickToBottom(monRecent);
        const prevTop = monRecent.scrollTop;
        if (recentLines.length === 0) {
          monInfo.textContent = "Sem eventos recentes.";
          monRecent.textContent = "";
        } else {
          monInfo.textContent = "Eventos recentes (" + String(recentLines.length) + ")";
          monRecent.textContent = recentLines.join("\n");
          if (stickToBottom) monRecent.scrollTop = monRecent.scrollHeight;
          else monRecent.scrollTop = prevTop;
        }
      } catch (err) {
        document.getElementById("monStatus").textContent = "Indisponivel";
        document.getElementById("monUpdated").textContent = new Date().toLocaleTimeString();
        document.getElementById("monInfo").textContent = "Falha ao atualizar monitoramento.";
      }
    }

    function closeDiagnosticsModal(ev){
      if (ev && ev.target && ev.target.id !== "diagModal") return;
      document.getElementById("diagModal").classList.add("hidden");
    }

    async function openDiagnosticsModal(){
      if (!hasPermission("logs:read")) {
        setStatus("Acesso negado: sem permissÃ£o para ver logs");
        return;
      }
      const modal = document.getElementById("diagModal");
      modal.classList.remove("hidden");
      
      const batchBtn = document.getElementById("diagBatchDelete");
      if (batchBtn) batchBtn.classList.toggle("hidden", !hasPermission("logs:delete"));

      await loadDiagnosticLogs();
    }

    function openDiagnosticFile(name){
      const url = "/api/diagnostics/log-file?name=" + encodeURIComponent(name);
      window.open(url, "_blank");
    }

    function escapeHtml(text){
      return String(text || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    }

    function diagHeaderRow(){
      return "<div class=\"diag-row diag-head\">" +
        "<div><input type=\"checkbox\" id=\"diag_select_all\" onchange=\"toggleSelectAllDiagnostic()\" /></div>" +
        "<div>Arquivo</div>" +
        "<div>data</div>" +
        "<div>Tamanho</div>" +
        "<div>AÃ§Ãµes</div>" +
      "</div>";
    }

    function selectedDiagnosticLogNames(){
      const out = [];
      const checks = document.querySelectorAll(".diag-row-check:checked");
      for (const el of checks) {
        const name = String(el.value || "").trim();
        if (name) out.push(name);
      }
      return out;
    }

    function toggleSelectAllDiagnostic(){
      const all = !!document.getElementById("diag_select_all").checked;
      const checks = document.querySelectorAll(".diag-row-check");
      for (const el of checks) el.checked = all;
    }

    async function deleteDiagnosticLog(name){
      if (!name) return;
      if (!confirm("Confirma excluir este log")) return;
      const res = await fetch("/api/diagnostics/log-delete?name=" + encodeURIComponent(name), {method: "POST"});
      const data = await res.json();
      setStatus(data.message || (data.ok ? "Arquivo removido" : "Falha ao remover arquivo"));
      await loadDiagnosticLogs();
    }

    async function deleteSelectedDiagnosticLogs(){
      const names = selectedDiagnosticLogNames();
      if (!names.length) {
        setStatus("Selecione ao menos um log");
        return;
      }
      if (!confirm("Confirma excluir os logs selecionados")) return;
      const res = await fetch("/api/diagnostics/log-delete-batch", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({names})
      });
      const data = await res.json();
      setStatus(data.message || (data.ok ? "Logs removidos" : "Falha ao remover logs"));
      await loadDiagnosticLogs();
    }

    async function loadDiagnosticLogs(){
      const list = document.getElementById("diagList");
      list.innerHTML = diagHeaderRow() + "<div class=\"diag-row\"><div></div><div>Carregando...</div><div></div><div></div><div></div></div>";

      try {
        const res = await fetch("/api/diagnostics/log-files");
        const data = await res.json();
        if (!res.ok) {
          list.innerHTML = diagHeaderRow() + "<div class=\"diag-row\"><div></div><div>Erro ao listar logs: " + escapeHtml(data.message || "desconhecido") + "</div><div></div><div></div><div></div></div>";
          return;
        }
        const files = Array.isArray(data.files) ? data.files : [];
        if (!files.length) {
          list.innerHTML = diagHeaderRow() + "<div class=\"diag-row\"><div></div><div>Nenhum log encontrado na pasta.</div><div></div><div></div><div></div></div>";
          return;
        }

        const rows = files.map((f) => {
          const fileName = String(f.name || "");
          const safeFile = fileName.replace(/\\/g, "\\\\").replace(/"/g, "&quot;");
          return "<div class=\"diag-row\">" +
            "<div><input type=\"checkbox\" class=\"diag-row-check\" value=\"" + escapeHtml(fileName) + "\" /></div>" +
            "<div><button type=\"button\" class=\"diag-open-link\" onclick=\"openDiagnosticFile(&quot;" + safeFile + "&quot;)\">" + escapeHtml(fileName) + "</button></div>" +
            "<div>" + escapeHtml(f.modified_at || "") + "</div>" +
            "<div>" + escapeHtml(formatBytes(f.size_bytes)) + "</div>" +
            "<div class=\"diag-actions\">" + 
            (hasPermission("logs:delete") ? ("<button type=\"button\" class=\"auth-action-btn auth-action-danger\" title=\"Excluir\" aria-label=\"Excluir\" onclick=\"deleteDiagnosticLog(&quot;" + safeFile + "&quot;)\">" + authActionIcon("delete") + "</button>") : "") +
            "</div>" +
          "</div>";
        }).join("");

        list.innerHTML = diagHeaderRow() + rows;
      } catch (err) {
        list.innerHTML = diagHeaderRow() + "<div class=\"diag-row\"><div></div><div>Falha ao carregar logs.</div><div></div><div></div><div></div></div>";
      }
    }

    setInterval(refreshLogs, 800);
    setInterval(refreshStatus, 1200);
    window.addEventListener("resize", () => {
      if (!isMobileLayout()) closeSidebarMobile();
    });
    document.addEventListener("keydown", (ev) => {
      if (ev.key === "Escape") {
        closeSidebarMobile();
        closeMonitoringModal();
        closeDiagnosticsModal();
        closeDashboardDetailsModal();
        closeSuccessMessagesModal();
        closeStatusDetailsModal();
        closeAuthEditModal();
        closeSolicitacaoEditModal();
        closeIDsGrupoEditModal();
        closeAppUserEditModal();
        closeRBACEditModal();
        closeUserMenu();
      }
    });
    document.getElementById("login_username").addEventListener("keydown", (ev) => {
      const loginErr = document.getElementById("login_error");
      if (loginErr) loginErr.textContent = "";
      if (ev.key === "Enter") {
        ev.preventDefault();
        loginAppUser();
      }
    });
    document.getElementById("login_password").addEventListener("keydown", (ev) => {
      const loginErr = document.getElementById("login_error");
      if (loginErr) loginErr.textContent = "";
      if (ev.key === "Enter") {
        ev.preventDefault();
        loginAppUser();
      }
    });
    document.getElementById("appuser_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchAppUsers();
      }
    });
    document.addEventListener("click", (ev) => {
      const wrap = document.querySelector(".user-menu-wrap");
      if (!wrap) return;
      if (!wrap.contains(ev.target)) closeUserMenu();
    });
    document.getElementById("auth_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchAuthUsers();
      }
    });
    document.getElementById("solicitacao_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchSolicitacoes();
      }
    });
    document.getElementById("my_solicitacao_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchMinhasSolicitacoes();
      }
    });
    document.getElementById("my_sol_status_filter").addEventListener("change", () => {
      searchMinhasSolicitacoes();
    });
    document.getElementById("my_sol_from").addEventListener("change", () => {
      searchMinhasSolicitacoes();
    });
    document.getElementById("my_sol_to").addEventListener("change", () => {
      searchMinhasSolicitacoes();
    });
    document.getElementById("sol_search_column").addEventListener("change", () => {
      searchSolicitacoes();
    });
    document.getElementById("sol_status_filter").addEventListener("change", () => {
      searchSolicitacoes();
    });
    document.getElementById("status").addEventListener("click", () => {
      openStatusDetailsModal();
    });
    document.getElementById("idsg_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchIDsGrupos();
      }
    });
    document.getElementById("idsg_search_column").addEventListener("change", () => {
      searchIDsGrupos();
    });
    document.getElementById("ga_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchGruposAtivos();
      }
    });
    document.getElementById("ga_search_column").addEventListener("change", () => {
      searchGruposAtivos();
    });
    document.getElementById("assembleia_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchAssembleias();
      }
    });
    document.getElementById("request_grupo").addEventListener("change", () => {
      syncSolicitarParcelasByGrupo(true);
    });
    document.getElementById("request_grupo").addEventListener("blur", () => {
      syncSolicitarParcelasByGrupo(true);
    });
    document.getElementById("request_grupo").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        syncSolicitarParcelasByGrupo(true);
      }
    });
    document.getElementById("request_grupo").addEventListener("input", () => {
      if (requestGrupoDebounceTimer) clearTimeout(requestGrupoDebounceTimer);
      requestGrupoDebounceTimer = setTimeout(() => syncSolicitarParcelasByGrupo(false), 220);
    });
    document.getElementById("sol_grupo").addEventListener("change", () => {
      syncSolicitacaoParcelasByGrupo(true);
    });
    document.getElementById("sol_grupo").addEventListener("blur", () => {
      syncSolicitacaoParcelasByGrupo(true);
    });
    document.getElementById("sol_grupo").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        syncSolicitacaoParcelasByGrupo(true);
      }
    });
    document.getElementById("sol_grupo").addEventListener("input", () => {
      if (solicitacaoGrupoDebounceTimer) clearTimeout(solicitacaoGrupoDebounceTimer);
      solicitacaoGrupoDebounceTimer = setTimeout(() => syncSolicitacaoParcelasByGrupo(false), 220);
    });
    document.getElementById("request_perc_lance").addEventListener("blur", () => {
      const el = document.getElementById("request_perc_lance");
      if (!el) return;
      el.value = formatPercentInputValue(el.value || "");
    });
    document.getElementById("sol_perc_lance").addEventListener("blur", () => {
      const el = document.getElementById("sol_perc_lance");
      if (!el) return;
      el.value = formatPercentInputValue(el.value || "");
    });
    document.getElementById("reserved_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchReservedCotas();
      }
    });
    document.getElementById("reserved_from").addEventListener("change", () => {
      searchReservedCotas();
    });
    document.getElementById("reserved_to").addEventListener("change", () => {
      searchReservedCotas();
    });
    document.getElementById("msg_search").addEventListener("keydown", (ev) => {
      if (ev.key === "Enter") {
        ev.preventDefault();
        searchManualNotifications();
      }
    });
    document.getElementById("msg_from").addEventListener("change", () => {
      searchManualNotifications();
    });
    document.getElementById("msg_to").addEventListener("change", () => {
      searchManualNotifications();
    });
    document.getElementById("msg_status").addEventListener("change", () => {
      searchManualNotifications();
    });
    setDashboardDefaultDates();
    setSolicitacoesDefaultDates();
    setMinhasSolicitacoesDefaultDates();
    setReservedDefaultDates();
    setMensagensDefaultDates();
    updateMobileHeaderOffset();
    window.addEventListener("resize", updateMobileHeaderOffset);
    window.addEventListener("orientationchange", updateMobileHeaderOffset);
    normalizeUIText();
    observeUTF8Fix();
    bindIDsGruposInfiniteScroll();
    decorateButtons();
    applySidebarState();
    initAllMasks();
    (async () => {
      try {
        await fetch("/api/app/logout", {method: "POST"});
      } catch (err) {}
      isAuthenticated = false;
      currentUserRole = "";
      window.currentUserRole = "";
      window.userPermissions = [];
      applyRolePermissions();
      showLoginOverlay(true);
      setStatus("Aguardando login");
      document.getElementById("login_username").focus();
    })();
// --- FASE 2: TOOLTIPS NATIVOS AUTOMATICOS ---
document.addEventListener("mouseover", function(e) {
  if (e.target && e.target.tagName && e.target.tagName.toLowerCase() === 'td') {
    if (e.target.offsetWidth < e.target.scrollWidth && !e.target.hasAttribute('title')) {
      e.target.setAttribute('title', e.target.innerText);
    }
  }
});
// --- FASE 3: RBAC (Controle de Acessos) ---
window.userPermissions = [];
window.currentUserRole = "";

function hasPermission(perm) {
    if (window.currentUserRole && window.currentUserRole.toLowerCase() === "admin") return true;
    return window.userPermissions.includes(perm);
}

function applyRBAC() {
    // Bloquear botoes de exclusao
    if (!hasPermission("solicitacoes:delete") && !hasPermission("users:delete") && !hasPermission("logs:delete")) {
        document.querySelectorAll('button[onclick*="delete"], button[onclick*="Delete"], button[onclick*="clearDatabaseTables"], button[onclick*="dropDatabaseTables"], button[onclick*="clearLogs"]').forEach(b => b.style.display = 'none');
    }
    
    // Bloqueio especÃ­fico para limpeza de logs
    if (!hasPermission("logs:delete")) {
        document.querySelectorAll('button[onclick*="clearLogs"]').forEach(b => b.style.display = 'none');
    }
    
    // Ocultar aba de usuarios se nao puder gerenciar
    const btnUser = document.querySelector('button[onclick="showConfigTab(\'appusers\')"]');
    if (btnUser) {
        btnUser.style.display = hasPermission("users:manage") ? '' : 'none';
    }

    // Ocultar aba de permissoes se nao for admin
    const btnPerm = document.querySelector('button[onclick="showConfigTab(\'rbac\')"]');
    if (btnPerm) {
        const isAdmin = window.currentUserRole && window.currentUserRole.toLowerCase() === "admin";
        btnPerm.style.display = isAdmin ? '' : 'none';
    }
}
let currentRBACMatrix = [];

async function loadRBACMatrix() {
    try {
        const res = await fetch("/api/rbac/matrix");
        const data = await res.json();
        if (!res.ok || !data.ok) throw new Error(data.message || "Erro ao carregar matriz RBAC");
        currentRBACMatrix = data.roles || [];
        renderRBACMatrix();
    } catch (err) {
        setStatus("Falha RBAC: " + err.message);
    }
}

function renderRBACMatrix() {
    const tbody = document.getElementById("rbac-tbody");
    if (!tbody) return;

    let tbodyHTML = "";
    currentRBACMatrix.forEach((role, roleIndex) => {
        tbodyHTML += `<tr>
            <td>${role.role_name}</td>
            <td>
                <button type="button" class="auth-action-btn auth-action-primary" onclick="openRBACEditModal(${roleIndex})" title="Editar PermissÃµes">
                    <svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                </button>
            </td>
        </tr>`;
    });
    tbody.innerHTML = tbodyHTML;
}

const permissionDescriptions = {
    "dashboard:read": "Acesso Ã  tela inicial e mÃ©tricas do Dashboard",
    "solicitacoes:read": "Visualizar a lista de solicitaÃ§Ãµes",
    "solicitacoes:create": "Criar novas solicitaÃ§Ãµes",
    "solicitacoes:edit": "Editar solicitaÃ§Ãµes existentes",
    "solicitacoes:delete": "Excluir solicitaÃ§Ãµes do sistema",
    "solicitacoes:print": "Imprimir solicitaÃ§Ãµes e relatÃ³rios",
    "cotas:reserve": "Reservar cotas",
    "cotas:export": "Exportar lista de cotas",
    "users:read": "Listar usuÃ¡rios no painel",
    "users:create": "Cadastrar novos usuÃ¡rios",
    "users:edit": "Editar dados de usuÃ¡rios",
    "users:delete": "Remover usuÃ¡rios do sistema",
    "roles:manage": "Gerenciar perfis e matriz de permissÃµes",
    "configs:manage": "Acessar configuraÃ§Ãµes gerais e de banco",
    "logs:read": "Visualizar logs do sistema",
    "logs:delete": "Limpar ou excluir logs do sistema",
    "audit:view": "Visualizar trilha de auditoria e histÃ³rico de aÃ§Ãµes"
};

function openRBACEditModal(roleIndex) {
    const role = currentRBACMatrix[roleIndex];
    if (!role) return;

    document.getElementById("rbacModalRoleIndex").value = roleIndex;
    document.getElementById("rbacModalTitle").textContent = `Editar PermissÃµes: ${role.role_name}`;

    const allPermsSet = new Set(Object.keys(permissionDescriptions));
    currentRBACMatrix.forEach(r => {
        if(r.permissions) r.permissions.forEach(p => allPermsSet.add(p));
    });
    const allPerms = Array.from(allPermsSet).sort();
    const rolePerms = new Set(role.permissions || []);

    const listDiv = document.getElementById("rbacModalList");
    let html = "";

    allPerms.forEach(p => {
        const isChecked = rolePerms.has(p) ? "checked" : "";
        const desc = permissionDescriptions[p] || "PermissÃ£o do sistema";
        html += `
        <label style="display: flex; align-items: flex-start; gap: 10px; cursor: pointer; padding: 5px 0;">
            <input type="checkbox" style="margin-top: 3px;" value="${p}" ${isChecked}>
            <div>
                <strong style="display: block; color: var(--text-color);">${p}</strong>
                <span style="font-size: 0.85rem; color: var(--text-muted);">${desc}</span>
            </div>
        </label>`;
    });

    listDiv.innerHTML = html;
    document.getElementById("rbacEditModal").classList.remove("hidden");
}

function closeRBACEditModal(ev) {
    if (ev && ev.target && ev.target.id !== "rbacEditModal") return;
    document.getElementById("rbacEditModal").classList.add("hidden");
}

function saveRBACEditModal() {
    const roleIndexStr = document.getElementById("rbacModalRoleIndex").value;
    if (!roleIndexStr) return;
    const idx = parseInt(roleIndexStr);

    const listDiv = document.getElementById("rbacModalList");
    const checkboxes = listDiv.querySelectorAll("input[type=checkbox]");
    
    const newPerms = [];
    checkboxes.forEach(chk => {
        if (chk.checked) newPerms.push(chk.value);
    });

    currentRBACMatrix[idx].permissions = newPerms;
    closeRBACEditModal();
    saveRBACMatrix(); // Persistir no servidor
}

async function saveRBACMatrix() {
    if (!confirm("Tem certeza que deseja salvar a nova matriz de permissoes? Os usuarios serao afetados no proximo login.")) return;

    try {
        const res = await fetch("/api/rbac/update", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ roles: currentRBACMatrix })
        });
        const data = await res.json();
        if (!res.ok || !data.ok) throw new Error(data.message || "Erro ao salvar");
        setStatus(data.message || "PermissÃµes salvas!");
        loadRBACMatrix();
    } catch (err) {
        setStatus("Falha: " + err.message);
    }
}

async function loadAuditLogs() {
    const q = document.getElementById("audit_search").value || "";
    const tbody = document.getElementById("audit_tbody");
    tbody.innerHTML = '<tr><td colspan="6">Carregando...</td></tr>';
    
    try {
        const res = await fetch(`/api/audit/list?q=${encodeURIComponent(q)}`);
        const data = await res.json();
        if(!data.ok) {
            tbody.innerHTML = `<tr><td colspan="6" style="color:red">Erro: ${data.message}</td></tr>`;
            return;
        }
        
        if(!data.logs || data.logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6">Nenhum log encontrado.</td></tr>';
            return;
        }
        
        tbody.innerHTML = data.logs.map(log => `
            <tr>
                <td>${formatDateTimeBR(log.created_at)}</td>
                <td><strong>${log.username}</strong></td>
                <td><span class="badge" style="background:var(--primary); color:white; padding:2px 6px; border-radius:4px; font-size:0.8em">${log.action}</span></td>
                <td>${log.entity || '-'}</td>
                <td>${log.entity_id || '-'}</td>
                <td style="font-size: 0.85em; color: var(--text-dim);">
                    ${log.before_state ? `<div style="color:var(--danger)">- ${log.before_state}</div>` : ''}
                    ${log.after_state ? `<div style="color:var(--success)">+ ${log.after_state}</div>` : ''}
                </td>
            </tr>
        `).join('');
    } catch(err) {
        tbody.innerHTML = `<tr><td colspan="6" style="color:red">Erro de conexÃ£o.</td></tr>`;
    }
}





