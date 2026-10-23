<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import {
  createTunnel,
  updateTunnel,
  type DomainItem,
  type NodeItem,
  type TunnelConfigUpdate,
  type TunnelItem,
  type UpdateTunnelRequest,
} from "@/services/center";
import {
  getLoadColor,
  getTypeMeta,
  isHTTPType,
  isNodeOnline,
  nodeSupportsType,
  tunnelTypes,
  type TunnelType,
} from "@/utils/tunnel";
import { isMobile, useFullscreenSystemBars } from "@/utils/platform";
import {
  describeService,
  describeServiceDetail,
  guessService,
  isLikelyService,
  listListeningPorts,
  type ListeningPort,
} from "@/utils/localService";

const props = defineProps<{
  mode: "create" | "edit";
  tunnel?: TunnelItem | null;
  nodes: NodeItem[];
  domains: DomainItem[];
}>();

const open = defineModel<boolean>({ required: true });

useFullscreenSystemBars(computed(() => open.value && isMobile));

const emit = defineEmits<{
  saved: [tunnel: TunnelItem | null, message: string];
}>();

const PROXY_PROTOCOL_OFF = "off";
const protocolItems = [
  { title: "TCP（默认）", value: "tcp" },
  { title: "KCP", value: "kcp" },
];
const proxyProtocolItems = [
  { title: "关闭", value: PROXY_PROTOCOL_OFF },
  { title: "v1", value: "v1" },
  { title: "v2", value: "v2" },
];

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | null>(null);
const submitting = ref(false);
const errorMessage = ref("");
const showAdvanced = ref(false);

const form = reactive({
  type: "tcp" as TunnelType,
  nodeId: null as number | null,
  remark: "",
  localIp: "127.0.0.1",
  localPort: "",
  remotePort: "",
  customDomain: "",
  protocol: "tcp",
  proxyProtocolVersion: PROXY_PROTOCOL_OFF,
  autoTLS: false,
  httpRedirect: false,
});

const isEdit = computed(() => props.mode === "edit");

// 新建引导：1 本地服务 → 2 类型与节点 → 3 访问方式
const wizardSteps = [
  { title: "本地服务", description: "选择要映射到公网的本机服务，或手动填写地址" },
  { title: "类型与节点", description: "选择隧道类型和承载流量的节点" },
  { title: "访问方式", description: "设置隧道名称与公网访问方式" },
];
const step = ref(1);
const bodyRef = ref<{ $el: HTMLElement } | null>(null);

watch(step, () => {
  bodyRef.value?.$el.scrollTo({ top: 0 });
});

// 本地监听端口扫描
const scanning = ref(false);
const scanError = ref("");
const listeningPorts = ref<ListeningPort[]>([]);
const showAllPorts = ref(false);
const portQuery = ref("");
// 选中的本地服务协议，用于限制可选的隧道类型；手动填写时为空
const localProtocol = ref<"tcp" | "udp" | "">("");
// 由所选服务自动填入的名称，用户未改动时随选择更新
let autoRemark = "";

// 局域网其他设备上可能有相同端口的服务，键需包含地址
const portKey = (item: ListeningPort) => `${item.protocol}/${item.port}/${item.local_ip}`;

const visiblePorts = computed(() => {
  const query = portQuery.value.trim().toLowerCase();
  return listeningPorts.value
    .filter((item) => {
      if (query) {
        return [String(item.port), item.process, item.title, describeService(item), item.protocol]
          .join(" ")
          .toLowerCase()
          .includes(query);
      }
      return showAllPorts.value || isLikelyService(item);
    })
    // 协议发现的服务（如 Minecraft 局域网世界）> 识别出的服务 > 其他，同组按端口排序
    .sort(
      (a, b) =>
        Number(Boolean(b.service)) - Number(Boolean(a.service)) ||
        Number(Boolean(guessService(b))) - Number(Boolean(guessService(a))) ||
        a.port - b.port,
    );
});

const hiddenPortCount = computed(
  () => listeningPorts.value.filter((item) => !isLikelyService(item)).length,
);

const selectedPortKey = computed(() => {
  const match = listeningPorts.value.find(
    (item) =>
      item.protocol === localProtocol.value &&
      String(item.port) === String(form.localPort).trim() &&
      item.local_ip === form.localIp.trim(),
  );
  return match ? portKey(match) : "";
});

const scanPorts = async () => {
  scanning.value = true;
  scanError.value = "";
  try {
    listeningPorts.value = await listListeningPorts();
  } catch (error) {
    scanError.value = error instanceof Error ? error.message : "扫描本地端口失败";
  } finally {
    scanning.value = false;
  }
};

const pickPort = (item: ListeningPort) => {
  form.localIp = item.local_ip || "127.0.0.1";
  form.localPort = String(item.port);
  localProtocol.value = item.protocol;

  // 按服务推荐隧道类型：UDP 服务只能用 UDP；网页服务在有已验证域名时推荐 HTTP
  const guess = guessService(item);
  if (item.protocol === "udp") {
    form.type = "udp";
  } else if (guess?.web && domainSuggestions.value.length > 0) {
    form.type = "http";
  } else {
    form.type = "tcp";
  }

  const name = (guess?.name || item.process || "").slice(0, 50);
  if (name && (!form.remark.trim() || form.remark === autoRemark)) {
    form.remark = name;
    autoRemark = name;
  }
};

// 手动修改地址后不再对应已选服务，解除协议限制
watch(
  () => [form.localIp, form.localPort],
  () => {
    if (localProtocol.value && !selectedPortKey.value) {
      localProtocol.value = "";
    }
  },
);

const isTypeDisabled = (type: TunnelType) => {
  if (isEdit.value || !localProtocol.value) return false;
  return localProtocol.value === "udp" ? type !== "udp" : type === "udp";
};

const nextStep = async () => {
  errorMessage.value = "";
  const result = await formRef.value?.validate();
  if (!result?.valid) return;
  step.value = Math.min(step.value + 1, wizardSteps.length);
};
const isHTTP = computed(() => isHTTPType(form.type));
const isHTTPS = computed(() => form.type === "https");

const resetForm = () => {
  errorMessage.value = "";
  showAdvanced.value = false;
  step.value = 1;
  showAllPorts.value = false;
  portQuery.value = "";
  localProtocol.value = "";
  autoRemark = "";
  const tunnel = props.tunnel;
  if (isEdit.value && tunnel) {
    const config = tunnel.config;
    form.type = (tunnel.type.toLowerCase() as TunnelType) || "tcp";
    form.nodeId = tunnel.node_id;
    form.remark = tunnel.remark || "";
    form.localIp = tunnel.local_ip || "";
    form.localPort = String(tunnel.local_port || "");
    form.remotePort = tunnel.remote_port > 0 ? String(tunnel.remote_port) : "";
    form.customDomain = tunnel.custom_domain || "";
    form.protocol = config?.protocol?.trim() || "tcp";
    form.proxyProtocolVersion =
      config?.proxy_protocol_version?.trim() || PROXY_PROTOCOL_OFF;
    form.autoTLS = Boolean(config?.auto_tls);
    form.httpRedirect = Boolean(config?.http_redirect);
    showAdvanced.value =
      form.protocol !== "tcp" || form.proxyProtocolVersion !== PROXY_PROTOCOL_OFF;
    return;
  }

  form.type = "tcp";
  form.nodeId = null;
  form.remark = "";
  form.localIp = "127.0.0.1";
  form.localPort = "";
  form.remotePort = "";
  form.customDomain = "";
  form.protocol = "tcp";
  form.proxyProtocolVersion = PROXY_PROTOCOL_OFF;
  form.autoTLS = false;
  form.httpRedirect = false;
  void scanPorts();
};

watch(open, (value) => {
  if (value) {
    resetForm();
  }
});

// 节点按 在线 > 支持当前类型 > 负载低 排序，不可用的节点置灰
const nodeItems = computed(() =>
  [...props.nodes]
    .map((node) => {
      const online = isNodeOnline(node);
      const supported = nodeSupportsType(node, form.type);
      return {
        node,
        online,
        supported,
        disabled: !online || !supported,
      };
    })
    .sort((a, b) => {
      if (a.disabled !== b.disabled) return a.disabled ? 1 : -1;
      const loadA = a.node.load ?? Number.POSITIVE_INFINITY;
      const loadB = b.node.load ?? Number.POSITIVE_INFINITY;
      return loadA - loadB;
    }),
);

type NodeOption = (typeof nodeItems.value)[number];

const filterNode = (_value: string, query: string, item?: { raw: NodeOption }) => {
  const node = item?.raw.node;
  if (!node) return false;
  return [node.name, String(node.id), node.sponsor, node.remark, node.region_code]
    .join(" ")
    .toLowerCase()
    .includes(query.toLowerCase());
};

const selectedNode = computed(
  () => props.nodes.find((node) => node.id === form.nodeId) ?? null,
);

// 切换类型后，若当前节点不支持则清空选择
watch(
  () => form.type,
  (type) => {
    if (isEdit.value || !selectedNode.value) return;
    if (!nodeSupportsType(selectedNode.value, type)) {
      form.nodeId = null;
    }
  },
);

const domainSuggestions = computed(() =>
  props.domains
    .filter((item) => item.is_verified)
    .map((item) => item.domain),
);

const portRule = (required: boolean) => (value: string) => {
  const text = String(value ?? "").trim();
  if (!text) return required ? "请输入端口" : true;
  const port = Number(text);
  return (Number.isInteger(port) && port >= 1 && port <= 65535) || "端口范围 1-65535";
};

const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
const ipRule = (value: string) => {
  const text = String(value ?? "").trim();
  if (!text) return "请输入本地 IP";
  // 中心端要求 IP 地址；IPv6 交由服务端校验
  return ipv4Pattern.test(text) || text.includes(":") || "请输入有效的 IP 地址";
};

const rules = {
  remark: [
    (v: string) => !!String(v ?? "").trim() || "请输入隧道名称",
    (v: string) => String(v ?? "").length <= 50 || "最多 50 个字符",
  ],
  node: [(v: number | null) => !!v || "请选择节点"],
  localIp: [ipRule],
  localPort: [portRule(true)],
  remotePort: [portRule(false)],
  domain: [
    (v: string) => !isHTTP.value || !!String(v ?? "").trim() || "请输入访问域名",
    (v: string) => !String(v ?? "").includes("*") || "不支持泛域名",
  ],
};

const buildUpdatePayload = (tunnel: TunnelItem): UpdateTunnelRequest => {
  const payload: UpdateTunnelRequest = {};
  const remark = form.remark.trim();
  if (remark !== (tunnel.remark || "").trim()) payload.remark = remark;

  const localIp = form.localIp.trim();
  if (localIp !== tunnel.local_ip) payload.local_ip = localIp;

  const localPort = Number(form.localPort.trim());
  if (localPort !== tunnel.local_port) payload.local_port = localPort;

  if (isHTTP.value) {
    const domain = form.customDomain.trim();
    if (domain !== (tunnel.custom_domain || "").trim()) payload.custom_domain = domain;
  }

  const config: TunnelConfigUpdate = {};
  const original = tunnel.config;
  if (form.protocol !== (original?.protocol?.trim() || "tcp")) {
    config.protocol = form.protocol;
  }
  const nextPPV =
    form.proxyProtocolVersion === PROXY_PROTOCOL_OFF ? "" : form.proxyProtocolVersion;
  if (nextPPV !== (original?.proxy_protocol_version?.trim() || "")) {
    config.proxy_protocol_version = nextPPV;
  }
  if (isHTTPS.value) {
    if (form.autoTLS !== Boolean(original?.auto_tls)) config.auto_tls = form.autoTLS;
    if (form.httpRedirect !== Boolean(original?.http_redirect)) {
      config.http_redirect = form.httpRedirect;
    }
  }
  if (Object.keys(config).length > 0) payload.config = config;
  return payload;
};

const handleSubmit = async () => {
  if (!isEdit.value && step.value < wizardSteps.length) {
    await nextStep();
    return;
  }
  errorMessage.value = "";
  const result = await formRef.value?.validate();
  if (!result?.valid) return;

  submitting.value = true;
  try {
    if (isEdit.value && props.tunnel) {
      const payload = buildUpdatePayload(props.tunnel);
      if (Object.keys(payload).length === 0) {
        open.value = false;
        emit("saved", null, "没有需要保存的修改");
        return;
      }
      const updated = await updateTunnel(props.tunnel.name, payload);
      open.value = false;
      emit("saved", updated, "隧道已更新");
      return;
    }

    const created = await createTunnel({
      node_id: form.nodeId ?? 0,
      type: form.type,
      local_ip: form.localIp.trim(),
      local_port: Number(form.localPort.trim()),
      remote_port: !isHTTP.value && form.remotePort.trim()
        ? Number(form.remotePort.trim())
        : undefined,
      custom_domain: isHTTP.value ? form.customDomain.trim() : undefined,
      remark: form.remark.trim(),
    });
    open.value = false;
    emit("saved", created, "隧道创建成功");
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "保存失败，请稍后重试";
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <v-dialog v-model="open" max-width="680" scrollable :fullscreen="isMobile" :persistent="submitting">
    <v-card class="md-dialog" :class="{ 'md-dialog--fullscreen': isMobile }">
      <!-- 标题 -->
      <div class="md-dialog-header">
        <h2 class="text-headline-small">{{ isEdit ? "编辑隧道" : "新建隧道" }}</h2>
        <p v-if="isEdit" class="text-body-medium md-on-surface-variant mt-2">
          类型、节点与远程端口创建后不可修改
        </p>
        <template v-else>
          <!-- 步骤指示：已完成 / 当前步骤高亮 -->
          <div class="wizard-steps mt-4" role="list">
            <div
              v-for="(item, index) in wizardSteps"
              :key="item.title"
              class="wizard-step"
              :class="{
                'wizard-step--active': index + 1 === step,
                'wizard-step--done': index + 1 < step,
              }"
              role="listitem"
            >
              <span class="wizard-step-bar" />
              <span class="text-label-medium">{{ index + 1 }}. {{ item.title }}</span>
            </div>
          </div>
          <p class="text-body-medium md-on-surface-variant mt-3">
            {{ wizardSteps[step - 1].description }}
          </p>
        </template>
      </div>

      <v-card-text ref="bodyRef" class="md-dialog-body" :class="{ 'md-dialog-body--wizard': !isEdit && !isMobile }">
        <v-alert
          v-if="errorMessage"
          type="error"
          variant="tonal"
          rounded="lg"
          density="compact"
          class="mb-4"
          closable
          @click:close="errorMessage = ''"
        >
          {{ errorMessage }}
        </v-alert>

        <v-form ref="formRef" @submit.prevent="handleSubmit">
          <!-- 步骤 1：选择本地服务 -->
          <template v-if="!isEdit && step === 1">
            <div class="d-flex align-center mb-3">
              <div class="md-section-label text-title-small mb-0">检测到的本地服务</div>
              <v-spacer />
              <v-btn
                variant="text"
                size="small"
                prepend-icon="fas fa-rotate"
                :loading="scanning"
                @click="scanPorts"
              >
                重新扫描
              </v-btn>
            </div>

            <v-text-field
              v-if="listeningPorts.length > 6"
              v-model="portQuery"
              placeholder="搜索端口 / 进程名"
              prepend-inner-icon="fas fa-magnifying-glass"
              density="compact"
              hide-details
              clearable
              class="mb-3"
            />

            <div v-if="scanning && listeningPorts.length === 0" class="port-empty md-on-surface-variant">
              <v-progress-circular indeterminate size="20" width="2" />
              <span class="text-body-medium">正在扫描本机监听端口…</span>
            </div>
            <div v-else-if="scanError" class="port-empty md-on-surface-variant">
              <span class="text-body-medium">{{ scanError }}，请手动填写本地地址</span>
            </div>
            <div v-else-if="visiblePorts.length === 0" class="port-empty md-on-surface-variant">
              <span class="text-body-medium">
                {{ portQuery ? "没有匹配的端口" : "未检测到常见服务，可手动填写本地地址" }}
              </span>
            </div>
            <div v-else class="port-list" role="radiogroup" aria-label="本地服务">
              <div
                v-for="item in visiblePorts"
                :key="portKey(item)"
                class="md-option port-option"
                :class="{ 'md-option--selected': selectedPortKey === portKey(item) }"
                role="radio"
                :aria-checked="selectedPortKey === portKey(item)"
                tabindex="0"
                @click="pickPort(item)"
                @keydown.enter.space.prevent="pickPort(item)"
              >
                <div class="md-icon-container port-icon">
                  <v-icon size="16">{{ guessService(item)?.icon || "fas fa-network-wired" }}</v-icon>
                </div>
                <div class="flex-grow-1 min-w-0">
                  <div class="text-body-large text-truncate">{{ describeService(item) }}</div>
                  <div class="text-body-small md-on-surface-variant text-truncate">
                    {{ describeServiceDetail(item) }}
                  </div>
                </div>
                <div class="port-meta">
                  <span class="text-title-medium md-mono">{{ item.port }}</span>
                  <span class="text-label-small md-on-surface-variant">{{ item.protocol.toUpperCase() }}</span>
                </div>
              </div>
            </div>
            <p v-if="isMobile" class="text-body-small md-on-surface-variant mt-3">
              Android 不允许应用读取完整的端口列表，这里仅探测常见的 TCP 端口和 Minecraft 服务器；其他 UDP 服务请手动填写。
            </p>
            <button
              v-if="!portQuery && hiddenPortCount > 0"
              type="button"
              class="md-expand text-label-large mt-3"
              @click="showAllPorts = !showAllPorts"
            >
              {{ showAllPorts ? "只显示常见服务" : `显示全部（另有 ${hiddenPortCount} 个系统 / 临时端口）` }}
              <v-icon size="12">{{ showAllPorts ? "fas fa-chevron-up" : "fas fa-chevron-down" }}</v-icon>
            </button>

            <v-divider class="my-4 md-divider" />
            <div class="md-section-label text-title-small">本地地址</div>
            <div class="local-grid">
              <v-text-field
                v-model="form.localIp"
                label="本地 IP"
                placeholder="127.0.0.1"
                :rules="rules.localIp"
              />
              <v-text-field
                v-model="form.localPort"
                label="本地端口"
                placeholder="例如：25565"
                :rules="rules.localPort"
                inputmode="numeric"
              />
            </div>
            <p class="text-body-small md-on-surface-variant">
              服务运行在局域网其他设备上时，填写该设备的 IP 即可。
            </p>
          </template>

          <!-- 隧道类型（编辑：只读展示；新建：步骤 2） -->
          <template v-if="isEdit || step === 2">
          <div class="md-section-label text-title-small">隧道类型</div>
          <div v-if="isEdit" class="d-flex align-center flex-wrap ga-2 mb-6">
            <v-chip variant="outlined" class="md-assist-chip" :prepend-icon="getTypeMeta(form.type).icon">
              {{ getTypeMeta(form.type).label }}
            </v-chip>
            <v-chip variant="outlined" class="md-assist-chip" prepend-icon="fas fa-server">
              {{ tunnel?.node_name || `节点 #${tunnel?.node_id}` }}
            </v-chip>
          </div>
          <v-item-group v-else v-model="form.type" mandatory class="type-grid mb-6">
            <v-item
              v-for="item in tunnelTypes"
              :key="item.value"
              v-slot="{ isSelected, toggle }"
              :value="item.value"
              :disabled="isTypeDisabled(item.value)"
            >
              <div
                class="md-option"
                :class="{
                  'md-option--selected': isSelected,
                  'md-option--disabled': isTypeDisabled(item.value),
                }"
                role="radio"
                :aria-checked="isSelected"
                :aria-disabled="isTypeDisabled(item.value)"
                tabindex="0"
                @click="!isTypeDisabled(item.value) && toggle?.()"
                @keydown.enter.space.prevent="!isTypeDisabled(item.value) && toggle?.()"
              >
                <div class="d-flex align-center ga-2">
                  <v-icon size="14">{{ item.icon }}</v-icon>
                  <span class="text-title-small">{{ item.label }}</span>
                  <v-spacer />
                  <v-icon v-if="isSelected" size="14" color="primary">fas fa-circle-check</v-icon>
                </div>
                <div class="text-body-small md-on-surface-variant mt-1">{{ item.description }}</div>
              </div>
            </v-item>
          </v-item-group>
          <p v-if="!isEdit && localProtocol" class="text-body-small md-on-surface-variant mt-n4 mb-6">
            所选本地服务使用 {{ localProtocol.toUpperCase() }}，已禁用不兼容的类型。
          </p>

          <!-- 节点 -->
          <template v-if="!isEdit && step === 2">
            <div class="md-section-label text-title-small">节点</div>
            <v-autocomplete
              v-model="form.nodeId"
              :items="nodeItems"
              :item-title="(item) => item.node.name"
              :item-value="(item) => item.node.id"
              :item-props="(item) => ({ disabled: item.disabled })"
              :rules="rules.node"
              :custom-filter="filterNode"
              label="选择节点"
              placeholder="搜索节点名 / ID / 赞助者"
              prepend-inner-icon="fas fa-server"
              no-data-text="暂无可用节点"
              class="mb-2"
            >
              <template #item="{ props: itemProps, item }">
                <v-list-item v-bind="itemProps" :title="undefined" class="py-2">
                  <div class="d-flex align-center ga-2">
                    <span
                      class="node-dot"
                      :class="item.online ? 'bg-success' : 'md-dot-offline'"
                    />
                    <span class="text-body-large">{{ item.node.name }}</span>
                    <span class="text-body-small md-on-surface-variant">#{{ item.node.id }}</span>
                    <v-spacer />
                    <span
                      v-if="item.node.load !== undefined && item.node.load !== null"
                      class="text-label-medium"
                      :class="`text-${getLoadColor(item.node.load)}`"
                    >
                      负载 {{ item.node.load.toFixed(0) }}%
                    </span>
                  </div>
                  <div class="d-flex align-center flex-wrap ga-1 mt-1 text-body-small md-on-surface-variant">
                    <span v-if="item.node.region_code">{{ item.node.region_code }}</span>
                    <span v-if="item.node.bandwidth">· {{ item.node.bandwidth }} Mbps</span>
                    <span v-if="item.node.traffic_ratio && item.node.traffic_ratio !== 1">
                      · {{ item.node.traffic_ratio }}x 流量
                    </span>
                    <span v-if="item.node.sponsor">· 赞助：{{ item.node.sponsor }}</span>
                    <v-chip v-if="item.node.need_kyc" size="x-small" color="warning" variant="tonal">
                      需实名
                    </v-chip>
                    <v-chip v-if="item.node.beian_required" size="x-small" color="info" variant="tonal">
                      建站需备案
                    </v-chip>
                    <v-chip v-if="!item.supported" size="x-small" color="error" variant="tonal">
                      不支持 {{ getTypeMeta(form.type).label }}
                    </v-chip>
                    <v-chip v-if="!item.online" size="x-small" variant="tonal">离线</v-chip>
                  </div>
                </v-list-item>
              </template>
            </v-autocomplete>
            <v-alert
              v-if="selectedNode?.need_kyc"
              type="warning"
              variant="tonal"
              rounded="lg"
              density="compact"
              class="mb-4"
            >
              该节点需要完成实名认证后才能创建隧道。
            </v-alert>
          </template>
          </template>

          <!-- 基础参数（编辑；新建：步骤 3） -->
          <template v-if="isEdit || step === 3">
          <div class="md-section-label text-title-small">基础参数</div>
          <v-text-field
            v-model="form.remark"
            label="隧道名称"
            placeholder="例如：MC 服务器"
            :rules="rules.remark"
            counter="50"
            class="mb-2"
          />
          <div class="mapping-grid">
            <div class="md-group">
              <div class="md-group-title text-label-large">
                <v-icon size="14">fas fa-desktop</v-icon>
                本地服务
              </div>
              <v-text-field
                v-model="form.localIp"
                label="本地 IP"
                placeholder="127.0.0.1"
                :rules="rules.localIp"
                class="mb-1"
              />
              <v-text-field
                v-model="form.localPort"
                label="本地端口"
                placeholder="例如：25565"
                :rules="rules.localPort"
                inputmode="numeric"
              />
            </div>

            <div class="mapping-arrow md-on-surface-variant">
              <v-icon size="18">fas fa-arrow-right-long</v-icon>
            </div>

            <div class="md-group">
              <div class="md-group-title text-label-large">
                <v-icon size="14">fas fa-cloud</v-icon>
                {{ isEdit ? tunnel?.node_name || "远程" : selectedNode?.name || "远程" }}
              </div>
              <v-combobox
                v-if="isHTTP"
                v-model="form.customDomain"
                :items="domainSuggestions"
                label="访问域名"
                placeholder="例如：mc.example.com"
                :rules="rules.domain"
                hint="需为已验证的域名或其子域名"
                persistent-hint
              />
              <v-text-field
                v-else
                v-model="form.remotePort"
                label="远程端口"
                :placeholder="isEdit ? '' : '留空自动分配'"
                :rules="rules.remotePort"
                :disabled="isEdit"
                inputmode="numeric"
                :hint="isEdit ? '远程端口创建后不可修改' : '留空由节点自动分配'"
                persistent-hint
              />
            </div>
          </div>

          <!-- 高级配置（编辑时可用） -->
          <template v-if="isEdit">
            <v-divider class="my-4 md-divider" />
            <button type="button" class="md-expand text-title-small" @click="showAdvanced = !showAdvanced">
              高级配置
              <v-icon size="12">{{ showAdvanced ? "fas fa-chevron-up" : "fas fa-chevron-down" }}</v-icon>
            </button>
            <v-expand-transition>
              <div v-show="showAdvanced" class="pt-4">
                <div class="advanced-grid">
                  <v-select
                    v-model="form.protocol"
                    :items="protocolItems"
                    label="连接协议"
                    hint="客户端与节点之间的传输协议"
                    persistent-hint
                  />
                  <v-select
                    v-model="form.proxyProtocolVersion"
                    :items="proxyProtocolItems"
                    label="Proxy Protocol"
                    hint="向本地服务传递真实客户端 IP"
                    persistent-hint
                  />
                </div>
                <div v-if="isHTTPS" class="md-switch-list mt-4">
                  <div class="md-switch-row">
                    <div>
                      <div class="text-body-large">Auto TLS</div>
                      <div class="text-body-medium md-on-surface-variant">自动签发并续期 HTTPS 证书</div>
                    </div>
                    <v-switch v-model="form.autoTLS" color="primary" inset hide-details />
                  </div>
                  <div class="md-switch-row">
                    <div>
                      <div class="text-body-large">HTTP 自动跳转 HTTPS</div>
                      <div class="text-body-medium md-on-surface-variant">需节点 frps ≥ 0.69.2</div>
                    </div>
                    <v-switch v-model="form.httpRedirect" color="primary" inset hide-details />
                  </div>
                </div>
              </div>
            </v-expand-transition>
          </template>
          <p v-else class="text-body-small md-on-surface-variant mt-4">
            连接协议、Proxy Protocol、Auto TLS 等高级配置可在创建后通过「编辑」设置。
          </p>
          </template>
        </v-form>
      </v-card-text>

      <!-- MD3 对话框操作：右对齐文字按钮 -->
      <div class="md-dialog-actions">
        <v-btn variant="text" color="primary" :disabled="submitting" @click="open = false">
          取消
        </v-btn>
        <v-spacer v-if="!isEdit" />
        <v-btn
          v-if="!isEdit && step > 1"
          variant="text"
          color="primary"
          :disabled="submitting"
          @click="step -= 1"
        >
          上一步
        </v-btn>
        <v-btn
          v-if="!isEdit && step < wizardSteps.length"
          variant="flat"
          color="primary"
          @click="nextStep"
        >
          下一步
        </v-btn>
        <v-btn
          v-else
          :variant="isEdit ? 'text' : 'flat'"
          color="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ isEdit ? "保存" : "创建" }}
        </v-btn>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
/* MD3 dialog：28dp 圆角、surface-container-high、24dp 内边距 */
.md-dialog-header {
  padding: 24px 24px 16px;
}

.md-dialog-body {
  padding: 8px 24px 0 !important;
}

/* 引导各步骤内容高度不同，固定最小高度避免对话框随步骤跳动 */
.md-dialog-body--wizard {
  min-height: 420px;
}

.md-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 24px 24px;
}

.md-on-surface-variant {
  color: var(--md-on-surface-variant);
}

.md-section-label {
  margin-bottom: 12px;
  color: rgb(var(--v-theme-on-surface));
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}

/* 可选卡片：未选中 outline-variant 描边，选中 secondary-container 填充 */
.md-option {
  position: relative;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--md-outline-variant);
  cursor: pointer;
  outline: none;
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease;
}

.md-option:hover {
  background: rgba(var(--v-theme-on-surface), 0.04);
}

.md-option:focus-visible {
  border-color: rgb(var(--v-theme-primary));
}

.md-option--selected,
.md-option--selected:hover {
  border-color: transparent;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-option--disabled {
  opacity: 0.38;
  cursor: not-allowed;
}

.md-option--disabled:hover {
  background: none;
}

/* 步骤指示：MD3 线性进度分段 */
.wizard-steps {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.wizard-step {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--md-on-surface-variant);
}

.wizard-step-bar {
  height: 4px;
  border-radius: 2px;
  background: var(--md-surface-container-highest);
}

.wizard-step--done .wizard-step-bar,
.wizard-step--active .wizard-step-bar {
  background: rgb(var(--v-theme-primary));
}

.wizard-step--active {
  color: rgb(var(--v-theme-primary));
}

.port-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.port-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
}

.port-icon {
  flex-shrink: 0;
}

.port-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex-shrink: 0;
}

.port-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 96px;
  padding: 16px;
  border-radius: 12px;
  border: 1px dashed var(--md-outline-variant);
  text-align: center;
}

.local-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 12px;
}

.min-w-0 {
  min-width: 0;
}

.md-assist-chip {
  border-color: var(--md-outline-variant) !important;
}

.mapping-grid {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 8px;
}

.mapping-arrow {
  display: flex;
  justify-content: center;
}

.md-group {
  height: 100%;
  padding: 12px 12px 4px;
  border-radius: 12px;
  background: var(--md-surface-container-highest);
}

.md-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--md-on-surface-variant);
}

.md-divider {
  border-color: var(--md-outline-variant);
  opacity: 1;
}

.md-expand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  border: none;
  background: none;
  color: rgb(var(--v-theme-primary));
  cursor: pointer;
}

.advanced-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.md-switch-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.md-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.md-switch-row .v-switch {
  flex: 0 0 auto;
}

.node-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.md-dot-offline {
  background: var(--md-outline);
}

@media (max-width: 600px) {
  .mapping-grid,
  .advanced-grid {
    grid-template-columns: 1fr;
  }

  .mapping-arrow {
    transform: rotate(90deg);
  }
}
</style>
