const app = document.getElementById("app");
const protocolSelect = document.getElementById("protocol");
const newGameButton = document.getElementById("new-game");
const openLibraryButton = document.getElementById("open-library");
const openHelpButton = document.getElementById("open-help");
const openGuideButton = document.getElementById("open-guide");
const overlay = document.getElementById("overlay");
const overlayTitle = document.getElementById("overlay-title");
const overlayContent = document.getElementById("overlay-content");
const closeOverlayButton = document.getElementById("close-overlay");
const pageShell = document.querySelector(".page-shell");
const hero = document.querySelector(".hero");

const SAVE_KEY = "codebreaker.session.save.v1";
const OPERATOR_KEY = "codebreaker.operator.name.v1";

let sessionId = null;
let currentState = null;
let overlayMode = null;
let hoveredDeckCardId = null;
let pinnedDeckCardId = null;
let selectedHandIndex = null;
let bootErrorMessage = "";
let operatorDraft = loadRememberedOperatorName();

const helpTerms = [
  ["生命", "你的主生命值，降到 0 本局结束。", "就是你的血条。"],
  ["内存", "也就是 MP。出牌要花内存。", "就是蓝条。"],
  ["指针", "Boss 战里可发动 Pointer Hijack。", "你的 Boss 大招次数。"],
  ["进度", "当前已清理节点数 / 全部节点数。", "这局打到哪了。"],
  ["负载", "高复杂度出牌会提高负载，影响下回合。", "打太猛，下一回合会卡。"],
  ["重构", "用于删牌，通常来自精英奖励。", "删牌次数。"],
  ["bits 数组", "敌人的关键属性都放在同一个数组里。", "敌人本质上是一串能被改写的数据。"],
  ["完整度", "敌人的生命值。", "就是敌人的血。"],
  ["倒计时", "敌人距离下次行动还剩几拍。", "数字越大越安全。"],
  ["DoS 锁牌", "后期敌人会锁定你一张手牌。", "这张牌本回合不能用。"],
  ["Stack Overflow", "连出多张牌后会触发回放与追击。", "连招打满后自动补伤害。"],
];

const simpleCardCopy = {
  "reverse_list.py": "把敌人的可移动 bits 整段翻过来。",
  "bubble_sort.sh": "重排前排 bits，还能拖慢敌人出手。",
  "binary_search.go": "敌人越有序，这张牌越容易精准拆攻击。",
  "backtrack.ts": "把战局回滚到上一回合。",
  "heap_alloc.cpp": "补蓝并摸牌。",
  "null_pointer.exe": "优先清护甲，没有护甲就直接打血。",
  "defrag_gc.cs": "清垃圾、减负载、回血。",
  "memory_leak.bin": "诅咒卡，留在手里会不断拖慢你。",
};

newGameButton.addEventListener("click", bootNewSession);
openLibraryButton.addEventListener("click", () => openOverlay(currentState ? "library" : "preLibrary"));
openHelpButton.addEventListener("click", () => openOverlay("help"));
openGuideButton.addEventListener("click", () => openOverlay("guide"));
closeOverlayButton.addEventListener("click", closeOverlay);
overlay.addEventListener("click", (event) => {
  if (event.target.dataset.closeOverlay === "true") closeOverlay();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") closeOverlay();
  if (event.key === "Enter" && document.activeElement?.id === "operator-input") {
    event.preventDefault();
    bootNewSession();
  }
});

document.addEventListener("mouseover", (event) => {
  const row = event.target.closest("[data-deck-card]");
  if (!row || pinnedDeckCardId) return;
  hoveredDeckCardId = row.dataset.deckCard;
  refreshDeckPreview();
});

document.addEventListener("input", (event) => {
  if (event.target.id === "operator-input") operatorDraft = event.target.value;
});

document.addEventListener("click", (event) => {
  const target = event.target.closest("[data-action]");
  if (!target) return;
  event.preventDefault();
  const action = target.dataset.action;
  if (action === "resume") return resumeSavedSession();
  if (action === "move") return postAction("/move", { direction: target.dataset.direction });
  if (action === "choose") return postAction("/choice", { index: Number(target.dataset.index) });
  if (action === "purge") return postAction("/purge", { index: Number(target.dataset.index) });
  if (action === "end-turn") return postAction("/combat/end");
  if (action === "hijack") return postAction("/combat/hijack");
  if (action === "restart") return bootNewSession();
  if (action === "toggle-deck-card") {
    const cardId = target.dataset.deckCard || null;
    pinnedDeckCardId = pinnedDeckCardId === cardId ? null : cardId;
    if (!pinnedDeckCardId) hoveredDeckCardId = null;
    else hoveredDeckCardId = cardId;
    return refreshDeckPreview();
  }
  if (action === "inspect-hand") {
    const index = Number(target.dataset.index);
    selectedHandIndex = selectedHandIndex === index ? null : index;
    return render();
  }
  if (action === "clear-hand") {
    selectedHandIndex = null;
    return render();
  }
  if (action === "play-selected" && selectedHandIndex != null) {
    return postAction("/combat/play", { index: Number(selectedHandIndex) });
  }
});

function loadRememberedOperatorName() {
  try {
    return localStorage.getItem(OPERATOR_KEY) || "";
  } catch {
    return "";
  }
}

function persistOperatorName(name) {
  const value = (name || "").trim();
  operatorDraft = value;
  try {
    if (value) localStorage.setItem(OPERATOR_KEY, value);
    else localStorage.removeItem(OPERATOR_KEY);
  } catch {}
}

function readOperatorName() {
  const input = document.getElementById("operator-input");
  return input?.value?.trim?.() || operatorDraft.trim() || loadRememberedOperatorName();
}

function loadSavedSession() {
  try {
    const raw = localStorage.getItem(SAVE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function autoSaveCurrentState() {
  if (!currentState || !sessionId) return;
  try {
    localStorage.setItem(
      SAVE_KEY,
      JSON.stringify({
        sessionId,
        operatorName: readOperatorName() || "anonymous",
        savedAt: new Date().toISOString(),
        sessionLabel: formatSessionBin(),
        snapshot: { state: currentState },
      }),
    );
  } catch {}
}

function clearSavedSession() {
  try {
    localStorage.removeItem(SAVE_KEY);
  } catch {}
}

function formatSessionBin() {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, "0");
  return `SESSION_${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}.bin`;
}

async function bootNewSession() {
  const operatorName = readOperatorName();
  if (!operatorName) {
    bootErrorMessage = "请输入 Operator ID 后再执行新线程。";
    return render();
  }
  persistOperatorName(operatorName);
  bootErrorMessage = "";
  newGameButton.disabled = true;
  newGameButton.textContent = "正在建立连接...";
  try {
    const response = await fetch("/api/session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ protocol: protocolSelect.value }),
    });
    currentState = normalizeState(await response.json());
    sessionId = currentState.sessionId;
    selectedHandIndex = null;
    hoveredDeckCardId = null;
    pinnedDeckCardId = null;
    autoSaveCurrentState();
    closeOverlay();
    render();
  } catch (error) {
    bootErrorMessage = `新会话创建失败：${safeText(error?.message, "网络连接异常")}`;
    render();
  } finally {
    newGameButton.disabled = false;
    newGameButton.textContent = "执行新线程";
  }
}

async function resumeSavedSession() {
  const save = loadSavedSession();
  if (!save?.sessionId) {
    bootErrorMessage = "没有找到可恢复会话。";
    return render();
  }
  bootErrorMessage = "正在重新建立连接...";
  render();
  try {
    const response = await fetch(`/api/session/${save.sessionId}`);
    if (!response.ok) {
      clearSavedSession();
      throw new Error("旧会话已失效，请重新开始。");
    }
    currentState = normalizeState(await response.json());
    sessionId = save.sessionId;
    protocolSelect.value = currentState.world?.protocol || protocolSelect.value;
    persistOperatorName(save.operatorName || "");
    hoveredDeckCardId = null;
    pinnedDeckCardId = null;
    selectedHandIndex = null;
    autoSaveCurrentState();
    render();
  } catch (error) {
    bootErrorMessage = `恢复失败：${safeText(error?.message, "无法重新建立连接")}`;
    render();
  }
}

function normalizeState(state) {
  if (!state) return null;
  state.messages = state.messages || [];
  state.cardPool = state.cardPool || [];
  state.player = state.player || {};
  state.player.artifacts = state.player.artifacts || [];
  state.player.lore = state.player.lore || [];
  state.player.loreEntries = state.player.loreEntries || [];
  state.player.deck = state.player.deck || [];
  state.world = state.world || {};
  state.world.nodes = state.world.nodes || [];
  state.world.availableMoves = state.world.availableMoves || {};
  if (state.battle) {
    state.battle.hand = state.battle.hand || [];
    state.battle.logs = state.battle.logs || [];
    state.battle.stack = state.battle.stack || [];
    state.battle.enemy = state.battle.enemy || {};
    state.battle.enemy.bits = state.battle.enemy.bits || [];
    state.battle.enemy.traits = state.battle.enemy.traits || [];
  }
  return state;
}

async function postAction(path, payload = null) {
  if (!sessionId) return;
  try {
    const response = await fetch(`/api/session/${sessionId}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: payload ? JSON.stringify(payload) : "{}",
    });
    const data = await response.json();
    if (!response.ok) {
      if (data?.state) {
        currentState = normalizeState(data.state);
        autoSaveCurrentState();
      }
      throw new Error(safeText(data?.error, "执行失败"));
    }
    currentState = normalizeState(data);
    if (currentState?.gameOver) clearSavedSession();
    else autoSaveCurrentState();
    selectedHandIndex = null;
    render();
  } catch (error) {
    render(safeText(error?.message, "执行失败"));
  }
}

function render(errorMessage = "") {
  removeFloatingThreadIndicator();
  if (!currentState) {
    applySceneState(false);
    return renderLanding(errorMessage || bootErrorMessage);
  }
  if (!currentState.battle) selectedHandIndex = null;
  const isBattle = Boolean(currentState.battle);
  app.className = isBattle ? "console-layout has-hand scene-battle" : "console-layout exploring-layout";
  app.innerHTML = "";
  applySceneState(isBattle);
  app.append(renderLeftColumn(errorMessage), renderCenterColumn(), renderRightColumn());
  if (isBattle) app.append(renderHandDock());
  bindRenderedActions();
  renderOverlay();
}

function renderLanding(errorMessage = "") {
  app.className = "landing-layout";
  const save = loadSavedSession();
  const operatorName = operatorDraft || loadRememberedOperatorName();
  app.innerHTML = `
    <section class="landing-console">
      <div class="landing-card">
        <div class="auth-console landing-terminal">
          <p class="eyebrow-line">AUTHENTICATION REQUIRED</p>
          <div class="auth-heading">KERNEL ACCESS GATE</div>
          <label class="terminal-auth"><span>Operator ID:</span><input id="operator-input" class="terminal-input" type="text" value="${escapeHtml(operatorName)}" maxlength="24" autocomplete="off" /></label>
          <div class="hero-actions landing-actions auth-actions">
            <button class="primary auth-button" id="inline-start">执行新线程</button>
            ${save ? `<button class="secondary auth-button" data-action="resume">EXECUTE ${escapeHtml(save.sessionLabel || formatSessionBin())}</button>` : ""}
          </div>
          <div class="landing-grid gameplay-grid">
            <article class="guide-card"><h3>先做什么</h3><p>输入代号后点击“执行新线程”，进入一条新的入侵线程。</p></article>
            <article class="guide-card"><h3>怎么玩</h3><p>沿着内核树推进，在战斗里重排敌人的 bits 数组，逐步改写 Root。</p></article>
            <article class="guide-card"><h3>建议先看</h3><p>右上角的“新手引导”和“术语帮助”会先把资源、负载、锁牌这些概念说清楚。</p></article>
          </div>
          ${errorMessage ? `<p class="micro-copy warning-copy">${escapeHtml(errorMessage)}</p>` : ""}
        </div>
      </div>
    </section>
  `;
  document.getElementById("inline-start")?.addEventListener("click", bootNewSession);
  bindRenderedActions();
  renderOverlay();
}

function removeFloatingThreadIndicator() {
  document.getElementById("thread-indicator")?.remove();
}

function applySceneState(isBattle) {
  pageShell?.classList.toggle("is-battle", Boolean(isBattle));
  hero?.classList.toggle("is-battle", Boolean(isBattle));
  document.body.classList.toggle("scene-battle", Boolean(isBattle));
  const totalLoad = (currentState?.player?.load || 0) + (currentState?.player?.pendingLoad || 0);
  const entropy = currentState?.battle?.enemy?.entropy || 0;
  document.body.classList.toggle("load-high", totalLoad >= 4);
  document.body.classList.toggle("entropy-high", entropy >= 7);
  document.body.classList.toggle("has-corruption", corruptionCount(currentState) > 0);
}

function renderLeftColumn(errorMessage) {
  const column = document.createElement("section");
  column.className = "left-column";
  column.append(moduleStatus(), moduleArtifacts(), moduleLogs(errorMessage));
  return column;
}

function moduleStatus() {
  const m = createModule("操作员状态", "status-module");
  const p = currentState.player || {};
  const w = currentState.world || {};
  const loadNow = p.load || 0;
  const pendingLoad = p.pendingLoad || 0;
  const totalLoad = loadNow + pendingLoad;
  const integrity = p.maxHp ? Math.round((p.hp / p.maxHp) * 100) : 0;
  const anomalies = [];
  if (pendingLoad > 0) anomalies.push(statusChip("待结算负载", `+${pendingLoad}`, "warning"));
  if ((currentState.battle?.hand || []).some((card) => card.locked)) anomalies.push(statusChip("锁牌", `${(currentState.battle.hand || []).filter((card) => card.locked).length} 张`, "danger"));
  if (corruptionCount(currentState) > 0) anomalies.push(statusChip("垃圾文件", `${corruptionCount(currentState)} 张`, "corrupt"));
  if ((currentState.battle?.enemy?.entropy || 0) >= 7) anomalies.push(statusChip("高熵", `${currentState.battle.enemy.entropy}`, "entropy"));
  m.querySelector(".module-body").innerHTML = `
    <div class="status-stack">
      <div class="thread-inline">
        <span>Current Thread</span>
        <strong>Node #${escapeHtml(String(w.currentNode ?? 0))}</strong>
        <em>Integrity ${escapeHtml(String(Math.max(0, integrity)))}%</em>
      </div>
      <div class="resource-bar-grid">
        ${resourceBar("生命", p.hp || 0, p.maxHp || 0, "hp", `${p.hp || 0}/${p.maxHp || 0}`)}
        ${resourceBar("内存", p.mp || 0, p.maxMp || 0, "mp", `${p.mp || 0}/${p.maxMp || 0}`)}
      </div>
      <div class="pressure-card ${loadSeverityClass(loadNow, pendingLoad)}">
        <div class="pressure-head"><span>系统负载</span><strong>${escapeHtml(String(loadNow))}${pendingLoad ? `<small> +${escapeHtml(String(pendingLoad))}</small>` : ""}</strong></div>
        <div class="pressure-meter"><span style="width:${Math.min(100, totalLoad * 18)}%"></span></div>
        <p class="pressure-copy">负载越高，下回合越容易出现少回蓝、少抽牌或线程卡顿。</p>
      </div>
      <div class="status-mini-grid">
        ${miniMetric("指针", p.pointers ?? 0, "pointer")}
        ${miniMetric("进度", safeText(w.completion, "0/0"), "progress")}
        ${miniMetric("重构", p.purgeCredits ?? 0, "refactor")}
      </div>
      <div class="protocol-brief">
        <p><strong>协议</strong>${escapeHtml(w.protocol === "postorder" ? "后序遍历" : "前序遍历")}</p>
        <p><strong>目标</strong>${escapeHtml(safeText(w.targetName, "当前没有剩余目标"))}</p>
        <p><strong>路径</strong>${escapeHtml(safeText(w.path, "-"))}</p>
      </div>
      <div class="anomaly-strip ${anomalies.length ? "is-active" : ""}">
        ${anomalies.length ? anomalies.join("") : `<span class="anomaly-idle">当前没有异常态，线程运行稳定。</span>`}
      </div>
    </div>
  `;
  return m;
}
function moduleArtifacts() {
  const m = createModule("遗物与碎片", "artifact-module");
  const items = currentState.player.artifacts || [];
  const entries = (currentState.player.loreEntries || []).slice(-2).reverse();
  const loreHtml = entries.length
    ? entries.map((entry) => `<p class="terminal-line"><strong>${escapeHtml(safeText(entry.title, "剧情碎片"))}</strong><br />${escapeHtml(safeText(entry.text, "暂无内容。"))}</p>`).join("")
    : (currentState.player.lore || []).slice(-2).map((line) => `<p class="terminal-line">${escapeHtml(safeText(line))}</p>`).join("");
  m.querySelector(".module-body").innerHTML = `
    <div class="compact-scroll">
      ${(items.length
        ? items.map((a) => `<p class="terminal-line"><strong>${escapeHtml(safeText(a.name, "未知遗物"))}</strong><br />${escapeHtml(safeText(a.desc, "暂无说明。"))}</p>`)
        : [`<p class="terminal-line">当前没有遗物。</p>`]
      ).join("")}
      ${loreHtml}
    </div>
  `;
  return m;
}

function moduleLogs(errorMessage) {
  const m = createModule("运行日志", "log-module");
  const lines = [...(errorMessage ? [errorMessage] : []), ...(currentState.messages || [])].slice(-12).reverse();
  m.querySelector(".module-body").innerHTML = `
    <div class="terminal-scroll">
      ${(lines.length ? lines : ["等待新的系统输出..."]).map((line) => `<p class="terminal-line">${escapeHtml(safeText(line))}</p>`).join("")}
    </div>
  `;
  return m;
}

function renderCenterColumn() {
  const column = document.createElement("section");
  column.className = "center-column";
  column.append(moduleMap(), moduleStage());
  return column;
}

function moduleMap() {
  const m = createModule("内核地图", `map-module${currentState.battle ? " battle-mini-map" : ""}`);
  const w = currentState.world;
  const levels = [
    { cls: "row-1", nodes: [{ id: 1, start: 7, span: 4 }] },
    { cls: "row-2", nodes: [{ id: 2, start: 3, span: 4 }, { id: 3, start: 11, span: 4 }] },
    { cls: "row-3", nodes: [{ id: 4, start: 1, span: 4 }, { id: 5, start: 5, span: 4 }, { id: 6, start: 9, span: 4 }, { id: 7, start: 13, span: 4 }] },
    { cls: "row-4", nodes: [{ id: 8, start: 1, span: 2 }, { id: 9, start: 3, span: 2 }, { id: 10, start: 5, span: 2 }, { id: 11, start: 7, span: 2 }, { id: 12, start: 9, span: 2 }, { id: 13, start: 11, span: 2 }, { id: 14, start: 13, span: 2 }, { id: 15, start: 15, span: 2 }] },
  ];
  const map = levels.map((level, index) => `${treeLevel(level, w)}${index < levels.length - 1 ? treeConnector(index + 1) : ""}`).join("");
  m.querySelector(".module-body").innerHTML = `
    <div class="tree-map-compact">${map}</div>
    <div class="map-legend">
      <span class="legend-chip is-current">当前线程</span>
      <span class="legend-chip is-cleared">已清理</span>
      <span class="legend-chip is-future">未进入</span>
      <span class="legend-chip is-target">协议目标</span>
    </div>
    <div class="map-footer">
      <div>
        <p class="micro-copy">当前位置：${escapeHtml(safeText(w.currentName, "-"))} / ${escapeHtml(kindLabel(w.currentKind))}</p>
        <p class="micro-copy">${escapeHtml(safeText(w.currentDesc, "当前节点暂无描述。"))}</p>
      </div>
      <div class="hero-actions">${moveButtons(w)}</div>
    </div>
  `;
  return m;
}

function treeLevel(level, world) {
  return `<div class="tree-level ${level.cls}">${level.nodes.map((node) => treeSlot(node, world)).join("")}</div>`;
}

function treeSlot(config, world) {
  const node = world.nodes.find((item) => item.index === config.id);
  return `<div class="tree-slot" style="grid-column:${config.start} / span ${config.span};">${treeNode(node, world)}</div>`;
}

function treeConnector(level) {
  const configs = {
    1: { horizontal: [25, 50], verticals: [50, 25, 75] },
    2: { horizontal: [12.5, 75], verticals: [25, 75, 12.5, 37.5, 62.5, 87.5] },
    3: { horizontal: [6.25, 87.5], verticals: [12.5, 37.5, 62.5, 87.5, 6.25, 18.75, 31.25, 43.75, 56.25, 68.75, 81.25, 93.75] },
  };
  const config = configs[level];
  return `<div class="tree-connector connector-${level}"><span class="tree-line horizontal" style="left:${config.horizontal[0]}%; width:${config.horizontal[1]}%;"></span>${config.verticals.map((pos) => `<span class="tree-line vertical" style="left:${pos}%;"></span>`).join("")}</div>`;
}

function treeNode(node, world) {
  if (!node) return `<div class="tree-node empty"></div>`;
  const cls = ["tree-node"];
  const available = Object.values(world.availableMoves || {}).includes(node.index);
  let stateLabel = "未进入";
  if (node.index === world.currentNode) {
    cls.push("active");
    stateLabel = "当前线程";
  } else if (node.cleared) {
    cls.push("cleared");
    stateLabel = "已清理";
  } else {
    cls.push("future");
    stateLabel = available ? "可前往" : "未进入";
  }
  if (available && node.index !== world.currentNode) cls.push("reachable");
  if (node.index === world.targetNode) {
    cls.push("target");
    stateLabel = node.index === world.currentNode ? stateLabel : "协议目标";
  }
  if (Math.abs(node.balance || 0) > 1) cls.push("unstable");
  return `<div class="${cls.join(" ")}"><span class="node-index">#${node.index}</span><strong>${escapeHtml(shorten(node.name, currentState.battle ? 10 : 16))}</strong><span>${escapeHtml(kindLabel(node.kind))}</span><em class="node-status">${escapeHtml(stateLabel)}</em></div>`;
}

function moveButtons(world) {
  const moves = world.availableMoves || {};
  const order = ["l", "r", "b"].filter((key) => moves[key] != null);
  if (!order.length) return `<span class="micro-copy">当前节点没有可移动方向。</span>`;
  return order
    .map((key) => {
      const node = world.nodes.find((item) => item.index === moves[key]);
      return `<button type="button" class="flat-action" data-action="move" data-direction="${key}"><strong>${escapeHtml(directionLabel(key))}</strong><small>${escapeHtml(node ? shorten(node.name, 12) : `#${moves[key]}`)}</small></button>`;
    })
    .join("");
}

function moduleStage() {
  const m = createModule(stageTitle(), "stage-module");
  const body = m.querySelector(".module-body");
  if (currentState.gameOver) body.innerHTML = gameOverContent();
  else if (currentState.pending) body.innerHTML = pendingContent();
  else if (currentState.battle) body.innerHTML = battleContent();
  else body.innerHTML = idleContent();
  return m;
}

function stageTitle() {
  if (currentState.gameOver) return "会话结果";
  if (currentState.pending) return { treasure: "遗物选择", reward: "脚本奖励", purge: "代码重构", boss_refactor: "Root 重构", boss_upgrade: "脚本升级" }[currentState.pending.kind] || "待处理事件";
  if (currentState.battle) return "敌方 bits 战场";
  return "控制台中枢";
}

function idleContent() {
  const w = currentState.world;
  return `<div class="idle-shell"><div class="guide-card fill-height"><p class="eyebrow-line">NODE STATUS</p><h3>${escapeHtml(safeText(w.currentName, "-"))}</h3><p>${escapeHtml(safeText(w.currentDesc, "当前节点暂无描述。"))}</p><p class="plain-copy">当前类型：${escapeHtml(kindLabel(w.currentKind))}。跟着协议目标推进，通常更容易拿到同步收益。</p></div></div>`;
}

function pendingContent() {
  const pending = currentState.pending;
  const action = pending.kind === "purge" ? "purge" : "choose";
  return `
    <div class="pending-shell">
      <div class="guide-card"><p>${escapeHtml(safeText(pending.message, "请选择一个后续动作。"))}</p></div>
      <div class="landing-grid gameplay-grid">
        ${(pending.choices || [])
          .map((choice) => `<button type="button" class="choice-chip" data-action="${action}" data-index="${choice.index}"><strong>${escapeHtml(safeText(choice.label, "未命名选项"))}</strong><span>${escapeHtml(safeText(choice.desc))}</span></button>`)
          .join("")}
      </div>
    </div>
  `;
}

function battleContent() {
  const b = currentState.battle;
  const enemy = b.enemy || {};
  const bits = enemy.bits || [];
  return `
    <div class="battle-shell battle-shell--hud">
      <div class="enemy-banner">
        <div>
          <p class="eyebrow-line">COMBAT TARGET</p>
          <h3>${escapeHtml(safeText(enemy.name, "Unknown Sentinel"))}</h3>
          <p class="micro-copy">${escapeHtml(safeText(enemy.intent, "敌方意图加载中"))}</p>
        </div>
        <div class="intent-chip ${enemyIntentTone(enemy.intent)}">${escapeHtml(safeText(enemy.intent, "待机"))}</div>
      </div>
      <div class="bits-arena">
        <div class="bits-arena-head">
          <span>Enemy Memory</span>
          <strong>bits[${bits.length}]</strong>
        </div>
        <div class="bits-grid">${renderEnemyBits(bits)}</div>
        <div class="bits-footnote">排序、翻转、查找与覆盖类脚本都会围绕这片数组战场生效。</div>
      </div>
      <div class="enemy-pills">
        ${enemyStatPill("完整度", enemy.integrity, "integrity")}
        ${enemyStatPill("攻击", enemy.attack, "attack")}
        ${enemyStatPill("护甲", enemy.armor, "armor")}
        ${enemyStatPill("倒计时", enemy.countdown, "countdown")}
        ${enemyStatPill("脚本值", enemy.script, "script")}
        ${enemyStatPill("熵值", enemy.entropy, `entropy ${entropySeverityClass(enemy.entropy || 0)}`)}
      </div>
      <div class="battle-support-grid">
        <section class="stage-card compact-card trait-card">
          <p class="sub-title">敌方特性</p>
          <div class="compact-scroll">
            ${(enemy.traits?.length ? enemy.traits : ["当前未检测到额外敌方特性。"]).map((trait) => `<p class="terminal-line">${escapeHtml(safeText(trait))}</p>`).join("")}
          </div>
        </section>
        <section class="stage-card compact-card flex-card output-card">
          <p class="sub-title">战斗输出</p>
          <div class="terminal-scroll combat-output-scroll">
            ${(b.logs?.length ? b.logs : ["等待新的战斗输出..."]).map((line) => `<p class="terminal-line">${escapeHtml(safeText(line))}</p>`).join("")}
          </div>
        </section>
      </div>
    </div>
  `;
}

function gameOverContent() {
  const ended = currentState.gameOver;
  const message = escapeHtml(String(ended.message ?? "本次会话结束。")).replaceAll("\n", "<br />");
  return `<div class="gameover-shell"><div class="guide-card fill-height"><p class="eyebrow-line">SESSION ${ended.won ? "SUCCESS" : "TERMINATED"}</p><h3>${ended.won ? "Root 已重写" : "会话已中断"}</h3><p class="gameover-copy">${message}</p><div class="hero-actions"><button class="primary" data-action="restart">重新执行</button></div></div></div>`;
}

function renderRightColumn() {
  const column = document.createElement("section");
  column.className = `right-column${currentState.battle ? " battle-column" : ""}`;
  const deckModule = createModule("卡牌蓝图", "deck-module");
  deckModule.querySelector(".module-body").innerHTML = deckShell();
  column.append(deckModule);
  if (currentState.battle) {
    const controlModule = createModule("脚本分析", "control-module");
    controlModule.querySelector(".module-body").innerHTML = battleControlShell();
    column.append(controlModule);
  }
  return column;
}

function deckShell() {
  const deck = currentState.player?.deck || [];
  const total = deck.reduce((sum, item) => sum + (item.count || 0), 0);
  return `
    <div class="deck-shell">
      <div class="deck-summary-row">
        <div class="deck-total">当前牌组 ${total} 张</div>
        <div class="micro-copy">点击条目固定详情</div>
      </div>
      <div class="deck-list">
        ${deck.length
          ? deck
              .map((item) => {
                const active = deckPreviewCardId() === item.card;
                const meta = cardMeta(item.card);
                return `<button type="button" class="deck-row ${active ? "active-preview" : ""} complexity-${complexitySlug(meta.complexity, meta.id)}" data-action="toggle-deck-card" data-deck-card="${escapeHtml(item.card)}"><span class="deck-icon">&lt;/&gt;</span><span class="deck-name">${escapeHtml(cardName(item.card))}</span><strong>x${item.count}</strong></button>`;
              })
              .join("")
          : `<p class="terminal-line">当前牌组为空。</p>`}
      </div>
      ${deckPreview()}
    </div>
  `;
}

function deckPreview() {
  const meta = cardMeta(deckPreviewCardId());
  if (!meta) {
    return `
      <div id="deck-preview" class="deck-preview visible empty">
        <div class="preview-head"><div><span class="preview-badge">deck://idle</span><h3>卡牌预览待机</h3></div></div>
        <p class="preview-copy">悬停可以快速预览，点击右侧条目会固定详情，方便战斗中随时查看而不丢焦点。</p>
      </div>
    `;
  }
  return `
    <div id="deck-preview" class="deck-preview visible complexity-${complexitySlug(meta.complexity, meta.id)}">
      <div class="preview-head">
        <div>
          <span class="preview-badge">${escapeHtml(safeText(meta.source, "card"))}</span>
          <h3>${escapeHtml(safeText(meta.name, meta.id))}</h3>
        </div>
        <div class="preview-cost">${escapeHtml(String(meta.cost || 0))} MP</div>
      </div>
      <p class="preview-meta">${escapeHtml(safeText(meta.complexity, "复杂度未知"))}</p>
      <p class="preview-copy">${escapeHtml(cardRuleText(meta))}</p>
    </div>
  `;
}

function battleControlShell() {
  const b = currentState.battle;
  const p = currentState.player;
  const hand = b.hand || [];
  const selected = selectedHandIndex != null ? hand.find((card) => card.index === selectedHandIndex) || null : null;
  const state = selectedCardState(selected, b, p);
  return `
    <div class="control-shell auxiliary-shell">
      <div class="hand-side-card ${selected ? "" : "empty"} ${state.slug ? `state-${state.slug}` : ""}">
        <div class="sub-title">${selected ? "当前脚本分析" : "等待选择"}</div>
        ${selected ? `<strong>${escapeHtml(safeText(selected.name, "未命名脚本"))}</strong><p class="micro-copy">${escapeHtml(safeText(selected.complexity, "复杂度未知"))} | ${escapeHtml(String(selected.cost || 0))} MP</p><p class="micro-copy">${escapeHtml(cardRuleText(selected))}</p>` : `<p class="micro-copy">点选下方手牌后，这里会显示脚本结构与当前执行状态。</p>`}
      </div>
      <div class="auxiliary-section">
        <div class="sub-title">执行栈</div>
        <div class="stack-list">
          ${(b.stack?.length ? b.stack.slice(-4).reverse() : ["当前栈为空。"]).map((item) => `<p class="terminal-line">${escapeHtml(safeText(item.name || item.card || item))}</p>`).join("")}
        </div>
      </div>
      <p class="micro-copy control-reason ${state.disabled ? "warning-copy" : ""}">${escapeHtml(state.reason)}</p>
    </div>
  `;
}

function renderHandDock() {
  const b = currentState.battle;
  const p = currentState.player;
  const hand = b.hand || [];
  const selected = selectedHandIndex != null ? hand.find((card) => card.index === selectedHandIndex) || null : null;
  const state = selectedCardState(selected, b, p);
  const center = hand.length > 1 ? (hand.length - 1) / 2 : 0;
  const dock = document.createElement("section");
  dock.className = "hand-dock";
  dock.innerHTML = `
    <div class="hand-dock-inner">
      <div class="hand-action-rail ${state.slug ? `state-${state.slug}` : "state-empty"}">
        <div class="selection-brief">
          <span class="selection-kicker">${selected ? "已选脚本" : "等待选择"}</span>
          <strong>${selected ? escapeHtml(safeText(selected.name, "未命名脚本")) : "点选一张手牌查看并准备执行"}</strong>
          <p class="selection-hint">${selected ? `${escapeHtml(safeText(selected.complexity, "复杂度未知"))} | ${escapeHtml(String(selected.cost || 0))} MP | ${escapeHtml(cardRuleText(selected))}` : "执行按钮已经收拢到手牌上方，不需要再把视线移到右侧。"}</p>
        </div>
        <div class="selection-actions">
          <button class="action primary-action" data-action="play-selected" ${state.disabled ? "disabled" : ""}>执行选中脚本</button>
          <button class="action secondary" data-action="end-turn">结束回合</button>
          <button class="action secondary" data-action="hijack" ${b.canHijack ? "" : "disabled"}>指针劫持</button>
          <button class="action secondary" data-action="clear-hand" ${selected ? "" : "disabled"}>取消选中</button>
        </div>
      </div>
      <div class="hand-container">
        ${hand.length
          ? hand
              .map((card, i) => {
                const angle = hand.length <= 1 ? 0 : ((i - center) / Math.max(center, 1)) * 5;
                const stateClass = selectedCardState(card, b, p);
                return `
                  <button
                    type="button"
                    class="hand-card ${selectedHandIndex === card.index ? "selected" : ""} complexity-${complexitySlug(card.complexity, card.id || card.name)} state-${stateClass.slug}"
                    data-action="inspect-hand"
                    data-index="${card.index}"
                    style="--slot:${i}; --fan-angle:${angle.toFixed(2)}deg;"
                  >
                    <span class="hand-cost">${escapeHtml(String(card.cost || 0))}</span>
                    <strong>${escapeHtml(safeText(card.name, "未命名脚本"))}</strong>
                    <span class="hand-meta">${escapeHtml(safeText(card.complexity, "复杂度未知"))}${card.unplayable ? " | 不可执行" : ""}${card.locked ? " | 已锁定" : ""}</span>
                    <small>${escapeHtml(shorten(cardRuleText(card), 54))}</small>
                  </button>
                `;
              })
              .join("")
          : `<div class="hand-empty-hint">本回合没有可执行手牌。</div>`}
      </div>
    </div>
  `;
  return dock;
}

function openOverlay(mode) {
  overlayMode = mode;
  renderOverlay();
}

function closeOverlay() {
  overlayMode = null;
  renderOverlay();
}

function renderOverlay() {
  if (!overlayMode) {
    overlay.classList.add("hidden");
    overlayTitle.textContent = "";
    overlayContent.innerHTML = "";
    return;
  }
  overlay.classList.remove("hidden");
  if (overlayMode === "library") {
    overlayTitle.textContent = "全卡图鉴";
    overlayContent.innerHTML = `<div class="guide-mini"><p>这里展示当前会话可读取到的全部卡牌信息。</p><p class="plain-copy">白话解释只放在图鉴和术语帮助里，方便第一次上手。</p></div><div class="overlay-card-grid">${(currentState.cardPool || []).map((card) => `<div class="overlay-card"><strong>${escapeHtml(safeText(card.name, card.id))}</strong><span>${escapeHtml(safeText(card.complexity, "复杂度未知"))} | ${escapeHtml(String(card.cost || 0))} MP | ${escapeHtml(safeText(card.source, "未知来源"))}</span><small>${escapeHtml(cardRuleText(card))}</small><small class="plain-copy">白话：${escapeHtml(simpleCardCopy[card.id] || "这张牌会直接改写敌人的 bits 结构。")}</small></div>`).join("")}</div>`;
    return;
  }
  if (overlayMode === "preLibrary") {
    overlayTitle.textContent = "全卡图鉴";
    overlayContent.innerHTML = `<div class="guide-mini"><p>先启动一局新会话，图鉴才会装载当前卡池。</p></div>`;
    return;
  }
  if (overlayMode === "help") {
    overlayTitle.textContent = "术语帮助";
    overlayContent.innerHTML = `<div class="guide-mini"><p><strong>怎么看页面</strong>：左边看资源和日志，中间看地图与 bits 战场，右边看当前牌组，底部战斗时看手牌与执行栏。</p></div><div class="help-list">${helpTerms.map(([term, desc, plain]) => `<article class="help-item"><h3>${escapeHtml(term)}</h3><p>${escapeHtml(desc)}</p><p class="plain-copy">白话：${escapeHtml(plain)}</p></article>`).join("")}</div>`;
    return;
  }
  overlayTitle.textContent = "新手引导";
  overlayContent.innerHTML = `<div class="guide-mini"><p><strong>一句话目标</strong>：沿着内核树推进，改写敌人的 bits，最后击败 Root 守卫。</p></div><div class="landing-grid gameplay-grid"><article class="guide-card"><h3>1. 开局</h3><p>输入 Operator ID 后点击“执行新线程”。</p></article><article class="guide-card"><h3>2. 地图</h3><p>地图是一棵树，你每次移动都是在遍历这棵树。</p></article><article class="guide-card"><h3>3. 战斗中心</h3><p>进入战斗后，中间最大的数组槽位就是敌人的 bits 战场，绝大多数算法都在这里生效。</p></article><article class="guide-card"><h3>4. 出牌</h3><p>先点选下方手牌，再直接在手牌上方执行；结束回合随时都能按，不需要先取消选中。</p></article><article class="guide-card"><h3>5. 右侧牌组</h3><p>右边的“卡牌蓝图”是你当前牌组。悬停快速看，点击条目可固定详情。</p></article><article class="guide-card"><h3>6. 图鉴与术语</h3><p>想看白话解释，请去“全卡图鉴”和“术语帮助”；战斗 HUD 里只保留更紧凑的技术说明。</p></article></div>`;
}

function createModule(title, extra = "") {
  const section = document.createElement("section");
  section.className = `console-module ${extra}`.trim();
  section.innerHTML = `<div class="module-head"><div class="module-title">${escapeHtml(title)}</div></div><div class="module-body"></div>`;
  return section;
}

function resourceBar(label, value, max, tone, text) {
  const safeMax = Math.max(1, Number(max) || 1);
  const percent = Math.max(0, Math.min(100, Math.round(((Number(value) || 0) / safeMax) * 100)));
  return `
    <div class="resource-bar-card ${tone}">
      <div class="resource-bar-head"><span>${escapeHtml(label)}</span><strong>${escapeHtml(text)}</strong></div>
      <div class="bar-track"><span class="bar-fill" style="width:${percent}%"></span></div>
    </div>
  `;
}

function statusChip(label, value, tone) {
  return `<span class="anomaly-chip ${tone}"><strong>${escapeHtml(label)}</strong><em>${escapeHtml(value)}</em></span>`;
}

function miniMetric(label, value, tone) {
  return `<div class="mini-metric ${tone}"><span>${escapeHtml(label)}</span><strong>${escapeHtml(String(value))}</strong></div>`;
}

function loadSeverityClass(load, pending) {
  const total = (load || 0) + (pending || 0);
  if (total >= 6) return "is-critical";
  if (total >= 4) return "is-high";
  if (total >= 2) return "is-warm";
  return "is-calm";
}

function renderEnemyBits(bits) {
  if (!Array.isArray(bits) || !bits.length) {
    return `<div class="bit-slot empty"><span class="bit-index">--</span><strong class="bit-value">NULL</strong><span class="bit-label">敌方内存未加载</span></div>`;
  }
  return bits
    .map((bit, index) => `
      <div class="bit-slot ${enemyBitClass(index, bit)}">
        <span class="bit-index">[${index}]</span>
        <strong class="bit-value">${escapeHtml(String(bit))}</strong>
        <span class="bit-label">${escapeHtml(enemyBitLabel(index))}</span>
      </div>
    `)
    .join("");
}

function enemyBitLabel(index) {
  const labels = ["攻击位", "护甲位", "倒计时位", "脚本位", "熵位", "校验位", "完整度镜像", "缓冲位"];
  return labels[index] || `扩展位 ${index}`;
}

function enemyBitClass(index, value) {
  const base = ["attack", "armor", "countdown", "script", "entropy", "checksum", "integrity", "buffer"][index] || "buffer";
  const hot = Number(value) >= 8 ? "hot" : Number(value) <= 0 ? "low" : "";
  return `${base} ${hot}`.trim();
}

function enemyStatPill(label, value, tone = "") {
  return `<div class="enemy-pill ${tone}"><span>${escapeHtml(label)}</span><strong>${escapeHtml(String(value ?? 0))}</strong></div>`;
}

function enemyIntentTone(intent) {
  const text = safeText(intent).toLowerCase();
  if (!text) return "neutral";
  if (text.includes("dos") || text.includes("锁") || text.includes("拒绝服务")) return "danger";
  if (text.includes("混淆") || text.includes("翻转") || text.includes("注入")) return "warning";
  if (text.includes("护甲") || text.includes("防御") || text.includes("排序")) return "accent";
  return "neutral";
}

function entropySeverityClass(entropy) {
  if (entropy >= 9) return "critical";
  if (entropy >= 6) return "high";
  if (entropy >= 3) return "mid";
  return "low";
}

function selectedCardState(selected, battle, player) {
  if (!selected) return { slug: "empty", disabled: true, reason: "先从下方手牌里选一张牌。" };
  if (battle?.turnLocked) return { slug: "locked", disabled: true, reason: "当前回合被冻结，只能结束回合。" };
  if (selected.locked) return { slug: "locked", disabled: true, reason: "这张牌已被锁定，本回合无法执行。" };
  if (selected.unplayable) return { slug: "corrupt", disabled: true, reason: "这是一张垃圾/诅咒牌，不能直接执行。" };
  if ((selected.cost || 0) > (player?.mp || 0)) return { slug: "starved", disabled: true, reason: "内存不足，无法执行这张脚本。" };
  return { slug: "ready", disabled: false, reason: "状态正常，点击“执行选中脚本”即可生效。" };
}

function deckPreviewCardId() {
  return pinnedDeckCardId || hoveredDeckCardId || null;
}

function refreshDeckPreview() {
  const preview = document.getElementById("deck-preview");
  if (preview) preview.outerHTML = deckPreview();
  document.querySelectorAll(".deck-row").forEach((row) => {
    row.classList.toggle("active-preview", row.dataset.deckCard === deckPreviewCardId());
    row.classList.toggle("is-pinned", row.dataset.deckCard === pinnedDeckCardId);
  });
}

function complexitySlug(complexity, id = "") {
  const text = `${complexity || ""} ${id || ""}`.toLowerCase();
  if (text.includes("memory_leak") || text.includes("null_pointer") || text.includes("overflow") || text.includes("诅咒")) return "corrupt";
  if (text.includes("o(1)") || text.includes("o(log n)")) return "constant";
  if (text.includes("o(n^2)") || text.includes("o(n*k)") || text.includes("branch^depth")) return "heavy";
  if (text.includes("o(n log n)")) return "hybrid";
  return "linear";
}

function corruptionCount(state) {
  if (!state) return 0;
  const handCorrupt = (state.battle?.hand || []).filter((card) => card.unplayable || /memory_leak|null_pointer|overflow|garbage|curse/i.test(card.id || card.name || "")).length;
  const deckCorrupt = (state.player?.deck || []).reduce((sum, item) => sum + (/memory_leak|null_pointer|overflow|garbage|curse/i.test(item.card || "") ? item.count || 0 : 0), 0);
  return handCorrupt + deckCorrupt;
}

function cardRuleText(card) {
  return safeText(card?.text, "脚本规则正在同步中，请结合复杂度与费用判断其用途。");
}

function cardMeta(id) {
  if (!id) return null;
  return currentState?.cardPool?.find((card) => card.id === id) || { id, name: id, cost: 0, complexity: "", text: "", source: "未知" };
}

function cardName(id) {
  return safeText(cardMeta(id)?.name, id);
}

function directionLabel(key) {
  return { l: "向左子节点", r: "向右子节点", b: "返回父节点" }[key] || key;
}

function kindLabel(kind) {
  return { start: "起点", combat: "战斗", rest: "休整", treasure: "宝物", boss: "Boss" }[kind] || kind || "未知";
}

function shorten(value, limit) {
  const text = safeText(value);
  return !text || text.length <= limit ? text : `${text.slice(0, limit - 1)}…`;
}

function safeText(value, fallback = "") {
  const text = String(value ?? "").replace(/\s+/g, " ").trim();
  if (!text || /^\?+$/.test(text)) return fallback;
  return text;
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

newGameButton.textContent = "执行新线程";

function bindRenderedActions() {
  document.querySelector(".deck-shell")?.addEventListener("mouseleave", () => {
    if (pinnedDeckCardId) return;
    hoveredDeckCardId = null;
    refreshDeckPreview();
  });
}

/* Battle HUD wireframe override */
const HERO_DEFAULT_HTML = hero?.firstElementChild?.innerHTML || "";
let deckDrawerOpen = false;

const battleHudUiHandler = (event) => {
  const trigger = event.target.closest("[data-ui-action]");
  if (!trigger) return;
  event.preventDefault();
  const action = trigger.dataset.uiAction;
  if (action === "toggle-deck-drawer") {
    deckDrawerOpen = !deckDrawerOpen;
    render();
  }
};

document.addEventListener("click", battleHudUiHandler);

function ensureSettingsButton() {
  if (!hero) return;
  let button = document.getElementById("open-settings");
  if (!button) {
    button = document.createElement("button");
    button.id = "open-settings";
    button.className = "secondary topbar-button";
    button.textContent = "设置";
    button.addEventListener("click", () => openOverlay("settings"));
    hero.querySelector(".hero-actions")?.append(button);
  }
}

function updateTopSystemBar() {
  if (!hero) return;
  const info = hero.firstElementChild;
  if (!currentState) {
    pageShell?.classList.remove("session-active");
    hero.classList.remove("system-bar", "battle-mode", "explore-mode");
    if (info) info.innerHTML = HERO_DEFAULT_HTML;
    return;
  }

  ensureSettingsButton();
  pageShell?.classList.add("session-active");
  hero.classList.add("system-bar");
  hero.classList.toggle("battle-mode", Boolean(currentState.battle));
  hero.classList.toggle("explore-mode", !currentState.battle);

  const p = currentState.player || {};
  const w = currentState.world || {};
  const entropy = currentState.battle?.enemy?.entropy ?? 0;
  const load = (p.load || 0) + (p.pendingLoad || 0);
  if (info) {
    info.innerHTML = `
      <div class="system-bar-core">
        <div class="system-brand">Code Breaker</div>
        <div class="system-metric-strip">
          ${topMetric("协议", w.protocol === "postorder" ? "后序" : "前序")}
          ${topMetric("节点", `#${w.currentNode ?? 0}`)}
          ${topMetric("HP", `${p.hp ?? 0}/${p.maxHp ?? 0}`, "hp")}
          ${topMetric("MP", `${p.mp ?? 0}/${p.maxMp ?? 0}`, "mp")}
          ${topMetric("负载", `${load}`, load >= 4 ? "load hot" : "load")}
          ${topMetric("熵", `${entropy}`, entropy >= 6 ? "entropy hot" : "entropy")}
        </div>
      </div>
    `;
  }

  newGameButton.textContent = currentState.battle ? "新会话" : "执行新线程";
}

function topMetric(label, value, extra = "") {
  return `<span class="top-metric ${extra}"><em>${escapeHtml(label)}</em><strong>${escapeHtml(String(value))}</strong></span>`;
}

function battleEntropyValue() {
  return currentState?.battle?.enemy?.entropy ?? 0;
}

function battleWarningTips() {
  const battle = currentState?.battle;
  const enemy = battle?.enemy || {};
  const tips = [];
  if ((enemy.countdown ?? 99) <= 1) tips.push("敌方将在下一拍行动，优先考虑控制或先压低攻击位。");
  if ((enemy.armor || 0) >= 5) tips.push("护甲较高，优先用穿透、削甲或直接改写类脚本。");
  if ((enemy.entropy || 0) >= 6) tips.push("熵值偏高，继续乱序可能触发更强副作用。");
  if ((battle?.hand || []).some((card) => card.locked)) tips.push("你有锁牌，本回合注意保留可执行资源。");
  if (!tips.length) tips.push("线程稳定，可以围绕敌方 bits 进行重排或定点打击。");
  return tips.slice(0, 3);
}

function battleRecommendation() {
  const battle = currentState?.battle;
  const enemy = battle?.enemy || {};
  const player = currentState?.player || {};
  if ((enemy.countdown ?? 99) <= 1 && player.mp >= 2) return "推荐优先拖慢倒计时，避免敌人立即反击。";
  if ((enemy.armor || 0) > (enemy.attack || 0)) return "推荐先削护甲，再打完整度，收益更高。";
  if ((player.load || 0) + (player.pendingLoad || 0) >= 4) return "推荐用低复杂度脚本缓一拍，避免下回合线程卡顿。";
  return "推荐观察 bits 排布后，再决定是翻转、排序还是直伤。";
}

function battleFeedbackLines() {
  return [...(currentState?.battle?.logs || [])].slice(-4).reverse();
}

function render(errorMessage = "") {
  removeFloatingThreadIndicator();
  updateTopSystemBar();
  if (!currentState) {
    applySceneState(false);
    return renderLanding(errorMessage || bootErrorMessage);
  }
  if (!currentState.battle) selectedHandIndex = null;
  const isBattle = Boolean(currentState.battle);
  app.className = isBattle ? "console-layout has-hand scene-battle wireframe-battle-layout" : "console-layout exploring-layout";
  app.innerHTML = "";
  applySceneState(isBattle);
  app.append(renderLeftColumn(errorMessage), renderCenterColumn(), renderRightColumn());
  if (isBattle) app.append(renderHandDock());
  bindRenderedActions();
  renderOverlay();
}

function renderLanding(errorMessage = "") {
  updateTopSystemBar();
  app.className = "landing-layout";
  const save = loadSavedSession();
  const operatorName = operatorDraft || loadRememberedOperatorName();
  app.innerHTML = `
    <section class="landing-console">
      <div class="landing-card">
        <div class="auth-console landing-terminal">
          <p class="eyebrow-line">AUTHENTICATION REQUIRED</p>
          <div class="auth-heading">KERNEL ACCESS GATE</div>
          <label class="terminal-auth"><span>Operator ID:</span><input id="operator-input" class="terminal-input" type="text" value="${escapeHtml(operatorName)}" maxlength="24" autocomplete="off" /></label>
          <div class="hero-actions landing-actions auth-actions">
            <button class="primary auth-button" id="inline-start">执行新线程</button>
            ${save ? `<button class="secondary auth-button" data-action="resume">EXECUTE ${escapeHtml(save.sessionLabel || formatSessionBin())}</button>` : ""}
          </div>
          <div class="landing-grid gameplay-grid">
            <article class="guide-card"><h3>先做什么</h3><p>输入代号后点击“执行新线程”，进入一条新的入侵线程。</p></article>
            <article class="guide-card"><h3>怎么玩</h3><p>沿着内核树推进，在战斗里重排敌人的 bits 数组，逐步改写 Root。</p></article>
            <article class="guide-card"><h3>建议先看</h3><p>右上角的“新手引导”和“术语帮助”会先把资源、负载、锁牌这些概念说清楚。</p></article>
          </div>
          ${errorMessage ? `<p class="micro-copy warning-copy">${escapeHtml(errorMessage)}</p>` : ""}
        </div>
      </div>
    </section>
  `;
  document.getElementById("inline-start")?.addEventListener("click", bootNewSession);
  bindRenderedActions();
  renderOverlay();
}

function applySceneState(isBattle) {
  pageShell?.classList.toggle("is-battle", Boolean(isBattle));
  pageShell?.classList.toggle("session-active", Boolean(currentState));
  hero?.classList.toggle("is-battle", Boolean(isBattle));
  document.body.classList.toggle("scene-battle", Boolean(isBattle));
  const totalLoad = (currentState?.player?.load || 0) + (currentState?.player?.pendingLoad || 0);
  const entropy = battleEntropyValue();
  document.body.classList.toggle("load-high", totalLoad >= 4);
  document.body.classList.toggle("entropy-high", entropy >= 7);
  document.body.classList.toggle("has-corruption", corruptionCount(currentState) > 0);
}

function renderLeftColumn(errorMessage) {
  const column = document.createElement("section");
  column.className = "left-column left-status-column";
  column.append(moduleStatus(), moduleLogs(errorMessage));
  return column;
}

function moduleStatus() {
  const m = createModule("操作员状态", "status-module compact-status-module");
  const p = currentState.player || {};
  const battle = currentState.battle;
  const entropy = battleEntropyValue();
  const anomalies = [];
  if ((battle?.hand || []).some((card) => card.locked)) anomalies.push(statusChip("锁牌", `${battle.hand.filter((card) => card.locked).length} 张`, "danger"));
  if (corruptionCount(currentState) > 0) anomalies.push(statusChip("垃圾文件", `${corruptionCount(currentState)} 张`, "corrupt"));
  if ((p.pendingLoad || 0) > 0) anomalies.push(statusChip("待结算负载", `+${p.pendingLoad}`, "warning"));

  const artifacts = (p.artifacts || []).slice(0, 5);
  m.querySelector(".module-body").innerHTML = `
    <div class="battle-status-stack">
      <section class="status-group resource-group">
        <p class="status-group-title">核心资源</p>
        ${resourceBar("生命", p.hp || 0, p.maxHp || 0, "hp", `${p.hp || 0}/${p.maxHp || 0}`)}
        ${resourceBar("内存", p.mp || 0, p.maxMp || 0, "mp", `${p.mp || 0}/${p.maxMp || 0}`)}
      </section>
      <section class="status-group pressure-group">
        <p class="status-group-title">压力资源</p>
        <div class="pressure-card ${loadSeverityClass(p.load || 0, p.pendingLoad || 0)}">
          <div class="pressure-head"><span>系统负载</span><strong>${escapeHtml(String((p.load || 0) + (p.pendingLoad || 0)))}</strong></div>
          <div class="pressure-meter"><span style="width:${Math.min(100, ((p.load || 0) + (p.pendingLoad || 0)) * 18)}%"></span></div>
        </div>
        <div class="entropy-card ${entropySeverityClass(entropy)}">
          <div class="entropy-head"><span>熵污染</span><strong>${escapeHtml(String(entropy))}</strong></div>
          <div class="entropy-meter"><span style="width:${Math.min(100, entropy * 10)}%"></span></div>
        </div>
      </section>
      <section class="status-group anomaly-group">
        <p class="status-group-title">异常状态</p>
        <div class="anomaly-strip ${anomalies.length ? "is-active" : ""}">
          ${anomalies.length ? anomalies.join("") : `<span class="anomaly-idle">当前没有锁牌、泄漏或垃圾文件异常。</span>`}
        </div>
      </section>
      <section class="status-group relic-group">
        <p class="status-group-title">遗物 / 被动效果</p>
        <div class="artifact-chip-list">
          ${artifacts.length ? artifacts.map((artifact) => `<span class="artifact-chip" title="${escapeHtml(safeText(artifact.desc, "暂无说明。"))}">${escapeHtml(safeText(artifact.name, "未知遗物"))}</span>`).join("") : `<span class="artifact-chip empty">暂无遗物</span>`}
        </div>
      </section>
    </div>
  `;
  return m;
}

function battleBitsSlot(bit, index) {
  const state = [];
  if (index < 2) state.push("核心位");
  else state.push("可移动");
  if (battleEntropyValue() >= 6 && index % 2 === 1) state.push("受噪点影响");
  return `
    <div class="bits-slot-card ${enemyBitClass(index, bit)}">
      <span class="bits-slot-index">#${String(index).padStart(2, "0")}</span>
      <strong class="bits-slot-value">${escapeHtml(String(bit))}</strong>
      <span class="bits-slot-state">${escapeHtml(state.join(" / "))}</span>
    </div>
  `;
}

function battleContent() {
  const battle = currentState.battle;
  const enemy = battle.enemy || {};
  const bits = enemy.bits || [];
  return `
    <div class="battle-wire-shell">
      <section class="enemy-headline-panel">
        <div class="enemy-core-icon">Σ</div>
        <div class="enemy-headline-copy">
          <p class="eyebrow-line">ENEMY PROCESS</p>
          <h3>${escapeHtml(safeText(enemy.name, "Unknown Sentinel"))}</h3>
          <p class="micro-copy">${escapeHtml(safeText(enemy.intent, "敌方意图加载中"))}</p>
        </div>
        <div class="enemy-headline-stats">
          <span>${escapeHtml(`倒计时 ${enemy.countdown ?? 0}`)}</span>
          <span>${escapeHtml(`攻击 ${enemy.attack ?? 0}`)}</span>
          <span>${escapeHtml(`护甲 ${enemy.armor ?? 0}`)}</span>
        </div>
      </section>
      <section class="bits-stage-panel">
        <div class="bits-stage-header">
          <div>
            <p class="eyebrow-line">BITS ARENA</p>
            <strong>敌方 bits[${bits.length}]</strong>
          </div>
          <div class="bits-stage-tags">
            <span class="arena-tag">完整度 ${escapeHtml(String(enemy.integrity ?? 0))}</span>
            <span class="arena-tag">脚本值 ${escapeHtml(String(enemy.script ?? 0))}</span>
            <span class="arena-tag entropy">熵 ${escapeHtml(String(enemy.entropy ?? 0))}</span>
          </div>
        </div>
        <div class="bits-main-lane">
          ${bits.length ? bits.map((bit, index) => battleBitsSlot(bit, index)).join("") : `<div class="bits-slot-card empty"><strong class="bits-slot-value">NULL</strong><span class="bits-slot-state">敌方数组未加载</span></div>`}
        </div>
      </section>
      <section class="combat-feedback-strip">
        ${(battleFeedbackLines().length ? battleFeedbackLines() : ["等待新的战斗反馈..."]).map((line) => `<span class="feedback-pill">${escapeHtml(safeText(line))}</span>`).join("")}
      </section>
    </div>
  `;
}

function contextPanelShell() {
  const battle = currentState.battle;
  const player = currentState.player;
  const hand = battle?.hand || [];
  const selected = selectedHandIndex != null ? hand.find((card) => card.index === selectedHandIndex) || null : null;
  const state = selectedCardState(selected, battle, player);
  const deck = currentState.player?.deck || [];

  if (!selected) {
    const enemy = battle?.enemy || {};
    return `
      <div class="context-shell idle-context-shell">
        <section class="context-block">
          <p class="context-title">敌方机制说明</p>
          <div class="context-list">
            ${(enemy.traits?.length ? enemy.traits : ["当前未检测到额外敌方机制。"]).map((trait) => `<p class="terminal-line">${escapeHtml(safeText(trait))}</p>`).join("")}
          </div>
        </section>
        <section class="context-block">
          <p class="context-title">本回合危险提示</p>
          <div class="context-list">
            ${battleWarningTips().map((tip) => `<p class="terminal-line warning-line">${escapeHtml(safeText(tip))}</p>`).join("")}
          </div>
        </section>
        <section class="context-block">
          <p class="context-title">当前推荐操作</p>
          <p class="context-body">${escapeHtml(battleRecommendation())}</p>
        </section>
        <section class="context-block deck-drawer-block ${deckDrawerOpen ? "open" : ""}">
          <div class="context-row">
            <p class="context-title">当前牌组抽屉</p>
            <button class="secondary mini-button" data-ui-action="toggle-deck-drawer">${deckDrawerOpen ? "收起牌组" : "展开牌组"}</button>
          </div>
          ${deckDrawerOpen ? `<div class="context-deck-list">${deck.length ? deck.map((item) => `<button type="button" class="deck-row ${deckPreviewCardId() === item.card ? "active-preview" : ""}" data-action="toggle-deck-card" data-deck-card="${escapeHtml(item.card)}"><span class="deck-icon">&lt;/&gt;</span><span class="deck-name">${escapeHtml(cardName(item.card))}</span><strong>x${item.count}</strong></button>`).join("") : `<p class="terminal-line">当前牌组为空。</p>`}</div>${deckPreview()}` : ""}
        </section>
      </div>
    `;
  }

  return `
    <div class="context-shell selected-context-shell">
      <section class="context-block selected-card-block state-${state.slug}">
        <p class="context-title">卡牌详情</p>
        <h3>${escapeHtml(safeText(selected.name, "未命名脚本"))}</h3>
        <div class="selected-card-meta">
          <span>${escapeHtml(safeText(selected.complexity, "复杂度未知"))}</span>
          <span>${escapeHtml(`${selected.cost || 0} MP`)}</span>
        </div>
        <p class="context-body">${escapeHtml(cardRuleText(selected))}</p>
        <p class="plain-copy">白话：${escapeHtml(simpleCardCopy[selected.id] || "这张牌会直接改写敌人的 bits 结构，请结合当前数组与资源判断时机。")}</p>
      </section>
      <section class="context-block">
        <p class="context-title">可执行判断</p>
        <p class="context-judge ${state.disabled ? "warning-copy" : "ready-copy"}">${escapeHtml(state.reason)}</p>
      </section>
      <section class="context-block deck-drawer-block ${deckDrawerOpen ? "open" : ""}">
        <div class="context-row">
          <p class="context-title">当前牌组抽屉</p>
          <button class="secondary mini-button" data-ui-action="toggle-deck-drawer">${deckDrawerOpen ? "收起牌组" : "展开牌组"}</button>
        </div>
        ${deckDrawerOpen ? `<div class="context-deck-list">${deck.length ? deck.map((item) => `<button type="button" class="deck-row ${deckPreviewCardId() === item.card ? "active-preview" : ""}" data-action="toggle-deck-card" data-deck-card="${escapeHtml(item.card)}"><span class="deck-icon">&lt;/&gt;</span><span class="deck-name">${escapeHtml(cardName(item.card))}</span><strong>x${item.count}</strong></button>`).join("") : `<p class="terminal-line">当前牌组为空。</p>`}</div>${deckPreview()}` : ""}
      </section>
    </div>
  `;
}

function renderRightColumn() {
  const column = document.createElement("section");
  column.className = `right-column ${currentState.battle ? "battle-context-column" : ""}`.trim();
  if (currentState.battle) {
    const contextModule = createModule("上下文面板", "context-module single-context-module");
    contextModule.querySelector(".module-body").innerHTML = contextPanelShell();
    column.append(contextModule);
    return column;
  }

  const deckModule = createModule("卡牌蓝图", "deck-module");
  deckModule.querySelector(".module-body").innerHTML = deckShell();
  column.append(deckModule);
  return column;
}

function renderHandDock() {
  const battle = currentState.battle;
  const player = currentState.player;
  const hand = battle?.hand || [];
  const selected = selectedHandIndex != null ? hand.find((card) => card.index === selectedHandIndex) || null : null;
  const state = selectedCardState(selected, battle, player);
  const center = hand.length > 1 ? (hand.length - 1) / 2 : 0;
  const dock = document.createElement("section");
  dock.className = "hand-dock hand-command-dock";
  dock.innerHTML = `
    <div class="hand-dock-inner">
      <div class="hand-action-rail ${state.slug ? `state-${state.slug}` : "state-empty"}">
        <div class="selection-brief compact-brief">
          <span class="selection-kicker">${selected ? "已选脚本" : "等待选择"}</span>
          <strong>${selected ? escapeHtml(safeText(selected.name, "未命名脚本")) : "请选择一张算法脚本"}</strong>
          <p class="selection-hint">${selected ? escapeHtml(shorten(cardRuleText(selected), 44)) : "先点选手牌，再在这里直接执行、结束回合或发动指针劫持。"}</p>
        </div>
        <div class="selection-actions compact-actions-row">
          <button class="action primary-action" data-action="play-selected" ${state.disabled ? "disabled" : ""}>执行此牌</button>
          <button class="action secondary" data-action="end-turn">结束回合</button>
          <button class="action secondary" data-action="hijack" ${battle.canHijack ? "" : "disabled"}>指针劫持</button>
          <button class="action secondary" data-action="clear-hand" ${selected ? "" : "disabled"}>取消选择</button>
        </div>
      </div>
      <div class="hand-container command-hand-container">
        ${hand.length ? hand.map((card, i) => {
          const angle = hand.length <= 1 ? 0 : ((i - center) / Math.max(center, 1)) * 5;
          const cardState = selectedCardState(card, battle, player);
          const summary = shorten(cardRuleText(card), 34);
          const tag = card.unplayable ? "垃圾" : card.locked ? "锁定" : cardState.disabled ? "受限" : "可执行";
          return `
            <button type="button" class="hand-card big-hand-card ${selectedHandIndex === card.index ? "selected" : ""} complexity-${complexitySlug(card.complexity, card.id || card.name)} state-${cardState.slug}" data-action="inspect-hand" data-index="${card.index}" style="--slot:${i}; --fan-angle:${angle.toFixed(2)}deg;">
              <span class="hand-cost">${escapeHtml(String(card.cost || 0))}</span>
              <strong>${escapeHtml(safeText(card.name, "未命名脚本"))}</strong>
              <span class="hand-meta">${escapeHtml(safeText(card.complexity, "复杂度未知"))}</span>
              <p class="hand-effect-line">${escapeHtml(summary)}</p>
              <span class="hand-tag">${escapeHtml(tag)}</span>
            </button>
          `;
        }).join("") : `<div class="hand-empty-hint">本回合没有可执行手牌。</div>`}
      </div>
    </div>
  `;
  return dock;
}

function renderOverlay() {
  if (!overlayMode) {
    overlay.classList.add("hidden");
    overlayTitle.textContent = "";
    overlayContent.innerHTML = "";
    return;
  }
  overlay.classList.remove("hidden");
  if (overlayMode === "settings") {
    overlayTitle.textContent = "设置";
    overlayContent.innerHTML = `<div class="guide-mini"><p>当前版本的设置主要用于说明战斗 HUD 的阅读方式。</p></div><div class="help-list"><article class="help-item"><h3>顶部状态栏</h3><p>显示协议、节点、HP、MP、负载与熵，是你战斗时最重要的系统状态摘要。</p></article><article class="help-item"><h3>右侧上下文面板</h3><p>未选牌时解释敌方机制；选中牌时解释这张牌能不能出，以及为什么。</p></article><article class="help-item"><h3>底部手牌主操作区</h3><p>执行、结束回合、指针劫持都贴近手牌，减少视线来回扫动。</p></article></div>`;
    return;
  }
  if (overlayMode === "library") {
    overlayTitle.textContent = "全卡图鉴";
    overlayContent.innerHTML = `<div class="guide-mini"><p>这里展示当前会话可读取到的全部卡牌信息。</p><p class="plain-copy">白话解释只放在图鉴和术语帮助里，方便第一次上手。</p></div><div class="overlay-card-grid">${(currentState.cardPool || []).map((card) => `<div class="overlay-card"><strong>${escapeHtml(safeText(card.name, card.id))}</strong><span>${escapeHtml(safeText(card.complexity, "复杂度未知"))} | ${escapeHtml(String(card.cost || 0))} MP | ${escapeHtml(safeText(card.source, "未知来源"))}</span><small>${escapeHtml(cardRuleText(card))}</small><small class="plain-copy">白话：${escapeHtml(simpleCardCopy[card.id] || "这张牌会直接改写敌人的 bits 结构。")}</small></div>`).join("")}</div>`;
    return;
  }
  if (overlayMode === "preLibrary") {
    overlayTitle.textContent = "全卡图鉴";
    overlayContent.innerHTML = `<div class="guide-mini"><p>先启动一局新会话，图鉴才会装载当前卡池。</p></div>`;
    return;
  }
  if (overlayMode === "help") {
    overlayTitle.textContent = "术语帮助";
    overlayContent.innerHTML = `<div class="guide-mini"><p><strong>怎么看页面</strong>：顶部看系统状态，左边看资源与异常，中间看敌方 bits 战场，右边看上下文解释，底部负责出牌。</p></div><div class="help-list">${helpTerms.map(([term, desc, plain]) => `<article class="help-item"><h3>${escapeHtml(term)}</h3><p>${escapeHtml(desc)}</p><p class="plain-copy">白话：${escapeHtml(plain)}</p></article>`).join("")}</div>`;
    return;
  }
  overlayTitle.textContent = "新手引导";
  overlayContent.innerHTML = `<div class="guide-mini"><p><strong>一句话目标</strong>：沿着内核树推进，改写敌人的 bits，最后击败 Root 守卫。</p></div><div class="landing-grid gameplay-grid"><article class="guide-card"><h3>1. 顶部先看什么</h3><p>顶部系统栏会告诉你协议、节点、HP、MP、负载和熵，先确认自己能不能贪大牌。</p></article><article class="guide-card"><h3>2. 中间在看什么</h3><p>中间不是普通怪物，而是敌方 bits 数组。你打出的排序、翻转、覆盖类脚本都会在这里产生结果。</p></article><article class="guide-card"><h3>3. 右边为什么变了</h3><p>右边现在是“解释器”。未选牌时帮你理解敌人，选牌后帮你判断这张牌能不能打。</p></article><article class="guide-card"><h3>4. 怎么出牌最快</h3><p>点选底部手牌后，直接在手牌上方点“执行此牌”；结束回合也在同一条操作带里。</p></article><article class="guide-card"><h3>5. 什么时候看牌组</h3><p>右侧上下文面板里有“当前牌组抽屉”，默认收起，需要时再展开，不会一直抢视线。</p></article><article class="guide-card"><h3>6. 探索和战斗的区别</h3><p>探索时地图更重要；战斗时地图会缩成 mini-map，让 bits 战场成为视觉中心。</p></article></div>`;
}

function bindRenderedActions() {
  document.querySelector(".context-deck-list")?.addEventListener("mouseleave", () => {
    if (pinnedDeckCardId) return;
    hoveredDeckCardId = null;
    refreshDeckPreview();
  });
}

render();
