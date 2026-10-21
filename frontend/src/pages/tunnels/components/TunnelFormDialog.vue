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
const isHTTP = computed(() => isHTTPType(form.type));
const isHTTPS = computed(() => form.type === "https");

const resetForm = () => {
  errorMessage.value = "";
  showAdvanced.value = false;
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
        <p class="text-body-medium md-on-surface-variant mt-2">
          {{
            isEdit
              ? "类型、节点与远程端口创建后不可修改"
              : "选择隧道类型和节点，把本地服务映射到公网"
          }}
        </p>
      </div>

      <v-card-text class="md-dialog-body">
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
          <!-- 隧道类型 -->
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
            >
              <div
                class="md-option"
                :class="{ 'md-option--selected': isSelected }"
                role="radio"
                :aria-checked="isSelected"
                tabindex="0"
                @click="toggle"
                @keydown.enter.space.prevent="toggle"
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

          <!-- 节点 -->
          <template v-if="!isEdit">
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

          <!-- 基础参数 -->
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
        </v-form>
      </v-card-text>

      <!-- MD3 对话框操作：右对齐文字按钮 -->
      <div class="md-dialog-actions">
        <v-btn variant="text" color="primary" :disabled="submitting" @click="open = false">
          取消
        </v-btn>
        <v-btn variant="text" color="primary" :loading="submitting" @click="handleSubmit">
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
