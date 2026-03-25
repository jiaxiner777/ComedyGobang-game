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

const SAVE_KEY = "codebreaker.session.save.v1";
const OPERATOR_KEY = "codebreaker.operator.name.v1";

let sessionId = null;
let currentState = null;
let overlayMode = null;
let hoveredDeckCardId = null;
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

document.addEventListener("mousemove", (event) => {
  const row = event.target.closest("[data-deck-card]");
  updateDeckPreview(row ? row.dataset.deckCard : null);
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
  try { return localStorage.getItem(OPERATOR_KEY) || ""; } catch { return ""; }
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
    localStorage.setItem(SAVE_KEY, JSON.stringify({
      sessionId,
      operatorName: readOperatorName() || "anonymous",
      savedAt: new Date().toISOString(),
      sessionLabel: formatSessionBin(),
      snapshot: { state: currentState },
    }));
  } catch {}
}

function clearSavedSession() {
  try { localStorage.removeItem(SAVE_KEY); } catch {}
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
    autoSaveCurrentState();
    closeOverlay();
    render();
  } catch (error) {
    bootErrorMessage = `新会话创建失败：${error.message}`;
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
    autoSaveCurrentState();
    render();
  } catch (error) {
    bootErrorMessage = `恢复失败：${error.message}`;
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
      throw new Error(data?.error || "???????");
    }
    currentState = normalizeState(data);
    if (currentState?.gameOver) clearSavedSession();
    else autoSaveCurrentState();
    selectedHandIndex = null;
    render();
  } catch (error) {
    render(error.message || "???????");
  }
}

function render(errorMessage = "") {
  if (!currentState) return renderLanding(errorMessage || bootErrorMessage);
  if (!currentState.battle) selectedHandIndex = null;
  app.className = currentState.battle ? "console-layout has-hand" : "console-layout exploring-layout";
  app.innerHTML = "";
  app.append(renderLeftColumn(errorMessage), renderCenterColumn(), renderRightColumn());
  if (currentState.battle) app.append(renderHandDock());
  bindRenderedActions();
  renderThreadIndicator();
  renderOverlay();
}

function renderLanding(errorMessage = "") {
  app.className = "landing-layout";
  const save = loadSavedSession();
  const operatorName = operatorDraft || loadRememberedOperatorName();
  app.innerHTML = `
    <section class="landing-console">
      <div class="landing-card">
        <div class="auth-console">
          <p class="eyebrow-line">AUTHENTICATION REQUIRED</p>
          <label class="terminal-auth"><span>Operator ID:</span><input id="operator-input" class="terminal-input" type="text" value="${escapeHtml(operatorName)}" maxlength="24" autocomplete="off" /></label>
          <div class="hero-actions landing-actions">
            <button class="primary" id="inline-start">执行新线程</button>
            ${save ? `<button class="secondary" data-action="resume">EXECUTE ${escapeHtml(save.sessionLabel || formatSessionBin())}</button>` : ""}
          </div>
          <div class="landing-grid gameplay-grid">
            <article class="guide-card"><h3>先做什么</h3><p>输入代号后点击“执行新线程”。</p></article>
            <article class="guide-card"><h3>怎么玩</h3><p>沿着地图推进，打赢战斗，逐步改写 Root。</p></article>
            <article class="guide-card"><h3>先看哪里</h3><p>建议先看右上角“新手引导”和“术语帮助”。</p></article>
          </div>
          ${errorMessage ? `<p class="micro-copy warning-copy">${escapeHtml(errorMessage)}</p>` : ""}
        </div>
      </div>
    </section>
  `;
  document.getElementById("inline-start")?.addEventListener("click", bootNewSession);
  bindRenderedActions();
  renderThreadIndicator();
  renderOverlay();
}
function renderThreadIndicator() {
  const existing = document.getElementById("thread-indicator");
  if (!currentState) return existing?.remove();
  const integrity = currentState.player?.maxHp ? Math.round((currentState.player.hp / currentState.player.maxHp) * 100) : 0;
  const el = existing || document.createElement("div");
  el.id = "thread-indicator";
  el.className = "thread-indicator";
  el.textContent = `Current Thread: Node #${currentState.world?.currentNode ?? 0} | Integrity: ${Math.max(0, integrity)}%`;
  if (!existing) document.body.append(el);
}

function renderLeftColumn(errorMessage) {
  const column = document.createElement("section");
  column.className = "left-column";
  column.append(moduleStatus(), moduleArtifacts(), moduleLogs(errorMessage));
  return column;
}

function moduleStatus() {
  const m = createModule("操作员状态", "status-module");
  const p = currentState.player;
  const w = currentState.world;
  m.querySelector(".module-body").innerHTML = `
    <div class="status-grid compact-grid">
      ${tile("生命", `${p.hp}/${p.maxHp}`)}${tile("内存", `${p.mp}/${p.maxMp}`)}${tile("指针", p.pointers ?? 0)}
      ${tile("进度", w.completion || "0/0")}${tile("负载", `${p.load ?? 0}${p.pendingLoad ? ` (+${p.pendingLoad})` : ""}`)}${tile("重构", p.purgeCredits ?? 0)}
    </div>
    <div class="guide-mini">
      <p><strong>协议：</strong>${escapeHtml(w.protocol === "postorder" ? "后序遍历" : "前序遍历")}</p>
      <p><strong>目标：</strong>${escapeHtml(w.targetName || "当前没有剩余目标")}</p>
      <p><strong>路径：</strong>${escapeHtml(w.path || "-")}</p>
    </div>
  `;
  return m;
}

function moduleArtifacts() {
  const m = createModule("遗物与增益", "artifact-module");
  const items = currentState.player.artifacts || [];
  const lore = (currentState.player.lore || []).slice(-2);
  m.querySelector(".module-body").innerHTML = `<div class="compact-scroll">${(items.length ? items.map((a) => `<p class="terminal-line"><strong>${escapeHtml(a.name)}</strong><br />${escapeHtml(a.desc)}</p>`) : [`<p class="terminal-line">当前没有遗物。</p>`]).join("")}${lore.map((line) => `<p class="terminal-line">${escapeHtml(line)}</p>`).join("")}</div>`;
  return m;
}

function moduleLogs(errorMessage) {
  const m = createModule("运行日志", "log-module");
  const lines = [...(errorMessage ? [errorMessage] : []), ...(currentState.messages || [])].slice(-10).reverse();
  m.querySelector(".module-body").innerHTML = `<div class="terminal-scroll">${(lines.length ? lines : ["等待新的系统输出..."]).map((line) => `<p class="terminal-line">${escapeHtml(line)}</p>`).join("")}</div>`;
  return m;
}

function renderCenterColumn() {
  const column = document.createElement("section");
  column.className = "center-column";
  column.append(moduleMap(), moduleStage());
  return column;
}
function moduleMap() {
  const m = createModule("内核地图", "map-module");
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
    <div class="map-legend"><span class="legend-chip is-current">当前线程</span><span class="legend-chip is-cleared">已清理节点</span><span class="legend-chip is-future">未进入节点</span><span class="legend-chip is-target">协议目标</span></div>
    <div class="map-footer"><div><p class="micro-copy">当前位置：${escapeHtml(w.currentName || "-")} / ${escapeHtml(kindLabel(w.currentKind))}</p><p class="micro-copy">${escapeHtml(w.currentDesc || "")}</p></div><div class="hero-actions">${moveButtons(w)}</div></div>
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
  return `<div class="${cls.join(" ")}"><span class="node-index">#${node.index}</span><strong>${escapeHtml(shorten(node.name, 16))}</strong><span>${escapeHtml(kindLabel(node.kind))}</span><em class="node-status">${escapeHtml(stateLabel)}</em></div>`;
}

function moveButtons(world) {
  const moves = world.availableMoves || {};
  const order = ["l", "r", "b"].filter((key) => moves[key] != null);
  if (!order.length) return `<span class="micro-copy">当前节点没有可移动方向。</span>`;
  return order.map((key) => {
    const node = world.nodes.find((item) => item.index === moves[key]);
    return `<button type="button" class="flat-action" data-action="move" data-direction="${key}"><strong>${escapeHtml(directionLabel(key))}</strong><small>${escapeHtml(node ? shorten(node.name, 14) : `#${moves[key]}`)}</small></button>`;
  }).join("");
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
  if (currentState.battle) return "战斗事件区";
  return "控制台中枢";
}

function idleContent() {
  const w = currentState.world;
  return `<div class="idle-shell"><div class="guide-card fill-height"><p class="eyebrow-line">NODE STATUS</p><h3>${escapeHtml(w.currentName || "-")}</h3><p>${escapeHtml(w.currentDesc || "当前节点暂无描述。")}</p><p class="plain-copy">当前类型：${escapeHtml(kindLabel(w.currentKind))}。跟着协议目标推进，通常更容易拿到同步收益。</p></div></div>`;
}

function pendingContent() {
  const pending = currentState.pending;
  const action = pending.kind === "purge" ? "purge" : "choose";
  return `<div class="pending-shell"><div class="guide-card"><p>${escapeHtml(pending.message || "请选择一个后续动作。")}</p></div><div class="landing-grid gameplay-grid">${(pending.choices || []).map((choice) => `<button type="button" class="choice-chip" data-action="${action}" data-index="${choice.index}"><strong>${escapeHtml(choice.label)}</strong><span>${escapeHtml(choice.desc || "")}</span></button>`).join("")}</div></div>`;
}

function battleContent() {
  const b = currentState.battle;
  return `
    <div class="battle-shell">
      <div class="battle-topline"><div><p class="eyebrow-line">COMBAT TARGET</p><h3>${escapeHtml(b.enemy.name || "Unknown")}</h3><p class="micro-copy">${escapeHtml(b.enemy.intent || "敌方意图加载中")}</p></div><div class="bits-terminal">${escapeHtml(formatBitsPreview(b.enemy.bits || []))}</div></div>
      <div class="status-grid battle-stat-row">${tile("完整度", b.enemy.integrity, true)}${tile("攻击", b.enemy.attack, true)}${tile("护甲", b.enemy.armor, true)}${tile("倒计时", b.enemy.countdown, true)}${tile("脚本值", b.enemy.script, true)}${tile("熵值", b.enemy.entropy, true)}</div>
      <div class="guide-card"><p><strong>敌方特性</strong></p><div class="compact-scroll">${(b.enemy.traits?.length ? b.enemy.traits : ["当前未检测到额外敌方特性。"]).map((t) => `<p class="terminal-line">${escapeHtml(t)}</p>`).join("")}</div></div>
      <div class="guide-card combat-output-scroll"><p><strong>战斗输出</strong></p><div class="terminal-scroll">${(b.logs || []).map((line) => `<p class="terminal-line">${escapeHtml(line)}</p>`).join("")}</div></div>
      <div class="hero-actions"><button class="action secondary" data-action="end-turn">结束回合</button><button class="action secondary" data-action="hijack" ${b.canHijack ? "" : "disabled"}>指针劫持</button></div>
    </div>
  `;
}

function gameOverContent() {
  const ended = currentState.gameOver;
  return `<div class="gameover-shell"><div class="guide-card fill-height"><p class="eyebrow-line">SESSION ${ended.won ? "SUCCESS" : "TERMINATED"}</p><h3>${ended.won ? "Root 已重写" : "会话已中断"}</h3><p>${escapeHtml(ended.message || "本次会话结束。")}</p><div class="hero-actions"><button class="primary" data-action="restart">重新执行</button></div></div></div>`;
}

function renderRightColumn() {
  const column = document.createElement("section");
  column.className = `right-column${currentState.battle ? " battle-column" : ""}`;
  const deckModule = createModule("卡牌蓝图", "deck-module");
  deckModule.querySelector(".module-body").innerHTML = deckShell();
  column.append(deckModule);
  if (currentState.battle) {
    const controlModule = createModule("出牌终端", "control-module");
    controlModule.querySelector(".module-body").innerHTML = battleControlShell();
    column.append(controlModule);
  }
  return column;
}

function deckShell() {
  const deck = currentState.player?.deck || [];
  return `<div class="deck-shell"><div class="micro-copy">当前牌组 ${deck.reduce((sum, item) => sum + (item.count || 0), 0)} 张</div><div class="deck-list">${deck.length ? deck.map((item) => `<button type="button" class="deck-row ${hoveredDeckCardId === item.card ? "active-preview" : ""}" data-deck-card="${escapeHtml(item.card)}"><span class="deck-icon">[]</span><span class="deck-name">${escapeHtml(cardName(item.card))}</span><strong>x${item.count}</strong></button>`).join("") : `<p class="terminal-line">当前牌组为空。</p>`}</div>${deckPreview()}</div>`;
}

function deckPreview() {
  const meta = hoveredDeckCardId ? cardMeta(hoveredDeckCardId) : null;
  if (!meta) return `<div id="deck-preview" class="deck-preview"></div>`;
  const plain = simpleCardCopy[meta.id] || "这张牌会直接改写敌人的 bits 结构，请结合费用和时机来使用。";
  return `<div id="deck-preview" class="deck-preview visible"><div class="preview-head"><div><span class="preview-badge">${escapeHtml(meta.source || "card")}</span><h3>${escapeHtml(meta.name || meta.id)}</h3></div><div class="preview-cost">${meta.cost || 0} MP</div></div><p class="preview-meta">${escapeHtml(meta.complexity || "")}</p><p class="preview-copy">${escapeHtml(meta.text || "")}</p><p class="plain-copy">白话：${escapeHtml(plain)}</p></div>`;
}

function battleControlShell() {
  const b = currentState.battle;
  const p = currentState.player;
  const hand = b.hand || [];
  const selected = selectedHandIndex != null ? hand.find((card) => card.index === selectedHandIndex) || null : null;
  const disabled = !selected || b.turnLocked || selected.locked || selected.unplayable || selected.cost > p.mp;
  const reason = !selected ? "先从下方手牌里选一张牌。" : b.turnLocked ? "当前回合被冻结，只能结束回合。" : selected.locked ? "这张牌已被锁定。" : selected.unplayable ? "这是一张不可执行的垃圾/诅咒牌。" : selected.cost > p.mp ? "内存不足。" : "点击“执行此牌”即可出牌。";
  return `<div class="control-shell"><div class="hand-side-card ${selected ? "" : "empty"}"><div class="sub-title">${selected ? "已选脚本" : "等待选择"}</div>${selected ? `<strong>${escapeHtml(selected.name)}</strong><p class="micro-copy">${escapeHtml(selected.complexity)} | ${selected.cost} MP</p><p class="micro-copy">${escapeHtml(selected.text)}</p>` : `<p class="micro-copy">点选一张手牌后，这里会显示详细说明。</p>`}</div><div class="control-actions"><button class="action primary-action" data-action="play-selected" ${disabled ? "disabled" : ""}>执行此牌</button><button class="action" data-action="end-turn">结束回合</button><button class="action" data-action="clear-hand" ${selected ? "" : "disabled"}>取消选中</button></div><p class="micro-copy control-reason${disabled ? " warning-copy" : ""}">${escapeHtml(reason)}</p></div>`;
}

function renderHandDock() {
  const b = currentState.battle;
  const hand = b.hand || [];
  const center = hand.length > 1 ? (hand.length - 1) / 2 : 0;
  const dock = document.createElement("section");
  dock.className = "hand-dock";
  dock.innerHTML = `<div class="hand-dock-inner"><div class="hand-container">${hand.length ? hand.map((card, i) => { const angle = hand.length <= 1 ? 0 : ((i - center) / Math.max(center, 1)) * 5; return `<button type="button" class="hand-card ${selectedHandIndex === card.index ? "selected" : ""} ${card.locked ? "locked" : ""}" data-action="inspect-hand" data-index="${card.index}" style="--slot:${i}; --fan-angle:${angle.toFixed(2)}deg;"><span class="hand-cost">${card.cost}</span><strong>${escapeHtml(card.name)}</strong><span class="hand-meta">${escapeHtml(card.complexity)}${card.unplayable ? " | 不可执行" : ""}${card.locked ? " | 已锁定" : ""}</span><small>${escapeHtml(shorten(card.text, 48))}</small></button>`; }).join("") : `<div class="hand-empty-hint">本回合没有可执行手牌。</div>`}</div></div>`;
  return dock;
}

function openOverlay(mode) { overlayMode = mode; renderOverlay(); }
function closeOverlay() { overlayMode = null; renderOverlay(); }

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
    overlayContent.innerHTML = `<div class="guide-mini"><p>这里展示当前会话可读取到的全部卡牌信息。</p><p class="plain-copy">白话解释只放在图鉴和术语帮助里，方便第一次上手。</p></div><div class="overlay-card-grid">${(currentState.cardPool || []).map((card) => `<div class="overlay-card"><strong>${escapeHtml(card.name)}</strong><span>${escapeHtml(card.complexity)} | ${card.cost} MP | ${escapeHtml(card.source || "未知来源")}</span><small>${escapeHtml(card.text)}</small><small class="plain-copy">白话：${escapeHtml(simpleCardCopy[card.id] || "这张牌会直接改写敌人的 bits 结构。")}</small></div>`).join("")}</div>`;
    return;
  }
  if (overlayMode === "preLibrary") {
    overlayTitle.textContent = "全卡图鉴";
    overlayContent.innerHTML = `<div class="guide-mini"><p>先启动一局新会话，图鉴才会装载当前卡池。</p></div>`;
    return;
  }
  if (overlayMode === "help") {
    overlayTitle.textContent = "术语帮助";
    overlayContent.innerHTML = `<div class="guide-mini"><p><strong>怎么看页面</strong>：左边看资源和日志，中间看地图与事件，右边看牌组，底部战斗时看手牌。</p></div><div class="help-list">${helpTerms.map(([term, desc, plain]) => `<article class="help-item"><h3>${escapeHtml(term)}</h3><p>${escapeHtml(desc)}</p><p class="plain-copy">白话：${escapeHtml(plain)}</p></article>`).join("")}</div>`;
    return;
  }
  overlayTitle.textContent = "新手引导";
  overlayContent.innerHTML = `<div class="guide-mini"><p><strong>一句话目标</strong>：沿着内核树推进，改写敌人的 bits，最后击败 Root 守卫。</p></div><div class="landing-grid gameplay-grid"><article class="guide-card"><h3>1. 开局</h3><p>输入 Operator ID 后点击“执行新线程”。</p></article><article class="guide-card"><h3>2. 地图</h3><p>地图是一棵树，你每次移动都是在遍历这棵树。</p></article><article class="guide-card"><h3>3. 战斗</h3><p>敌人不是普通怪，而是一串可被排序、翻转和查找的数据。</p></article><article class="guide-card"><h3>4. 出牌</h3><p>先选牌，再点“执行此牌”；想结束回合随时都可以直接结束。</p></article><article class="guide-card"><h3>5. 负载</h3><p>高复杂度牌会让你下回合变卡，别只顾着贪大招。</p></article><article class="guide-card"><h3>6. 牌组与图鉴</h3><p>右侧卡牌蓝图是你当前牌组，全卡图鉴是当前能查看到的全部卡牌说明。</p></article></div>`;
}

function createModule(title, extra = "") {
  const section = document.createElement("section");
  section.className = `console-module ${extra}`.trim();
  section.innerHTML = `<div class="module-head"><div class="module-title">${escapeHtml(title)}</div></div><div class="module-body"></div>`;
  return section;
}

function tile(label, value, compact = false) { return `<div class="status-tile ${compact ? "compact-tile" : ""}"><span>${escapeHtml(label)}</span><strong>${escapeHtml(String(value))}</strong></div>`; }
function cardMeta(id) { return currentState?.cardPool?.find((card) => card.id === id) || { id, name: id, cost: 0, complexity: "", text: "", source: "未知" }; }
function cardName(id) { return cardMeta(id).name || id; }
function updateDeckPreview(id) { hoveredDeckCardId = id || null; const preview = document.getElementById("deck-preview"); if (preview) preview.outerHTML = deckPreview(); document.querySelectorAll(".deck-row").forEach((row) => row.classList.toggle("active-preview", row.dataset.deckCard === hoveredDeckCardId)); }
function directionLabel(key) { return { l: "向左子节点", r: "向右子节点", b: "返回父节点" }[key] || key; }
function kindLabel(kind) { return { start: "起点", combat: "战斗", rest: "休整", treasure: "宝物", boss: "Boss" }[kind] || kind || "未知"; }
function shorten(value, limit) { return !value || value.length <= limit ? (value || "") : `${value.slice(0, limit - 1)}…`; }
function formatBitsPreview(bits, limit = 8) {
  if (!Array.isArray(bits) || bits.length === 0) return "bits[0] = []";
  const shown = bits.slice(0, limit).join(", ");
  const rest = bits.length > limit ? `, ... +${bits.length - limit}` : "";
  return `bits[${bits.length}] = [${shown}${rest}]`;
}

function escapeHtml(value) { return String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;"); }

newGameButton.textContent = "执行新线程";
render();

function bindRenderedActions() {}

