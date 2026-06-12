import { defineStore } from "pinia";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import {
  cancelInstallOrUpdateFrpc,
  installOrUpdateFrpc,
  type FrpcInstallResult,
} from "@/services/frpc";

const PROGRESS_EVENT = "frpc_install_progress";

export type InstallPhase =
  | "idle"
  | "resolving"
  | "downloading"
  | "verifying"
  | "extracting"
  | "done";

type ProgressPayload = {
  phase: InstallPhase;
  downloaded: number;
  total: number;
  percent: number;
};

export const useFrpcInstallStore = defineStore("frpcInstall", {
  state: () => ({
    installing: false,
    canceling: false,
    runningPromise: null as Promise<FrpcInstallResult> | null,
    phase: "idle" as InstallPhase,
    downloaded: 0,
    total: 0,
    percent: 0,
  }),
  getters: {
    // True while downloading with an unknown total size — render an
    // indeterminate progress bar in that case.
    indeterminate(state): boolean {
      return state.installing && state.phase !== "downloading"
        ? true
        : state.total <= 0;
    },
  },
  actions: {
    resetProgress() {
      this.phase = "idle";
      this.downloaded = 0;
      this.total = 0;
      this.percent = 0;
    },

    async startInstall(): Promise<FrpcInstallResult> {
      if (this.runningPromise) {
        return this.runningPromise;
      }

      this.installing = true;
      this.canceling = false;
      this.resetProgress();
      this.phase = "resolving";

      EventsOn(PROGRESS_EVENT, (payload: ProgressPayload) => {
        if (!payload) {
          return;
        }
        this.phase = payload.phase ?? this.phase;
        this.downloaded = payload.downloaded ?? 0;
        this.total = payload.total ?? 0;
        this.percent = payload.percent ?? 0;
      });

      this.runningPromise = (async () => {
        try {
          return await installOrUpdateFrpc();
        } finally {
          EventsOff(PROGRESS_EVENT);
          this.installing = false;
          this.canceling = false;
          this.runningPromise = null;
          this.resetProgress();
        }
      })();

      return this.runningPromise;
    },

    async cancelInstall(): Promise<void> {
      if (!this.runningPromise || this.canceling) {
        return;
      }

      this.canceling = true;
      try {
        await cancelInstallOrUpdateFrpc();
      } catch (error) {
        this.canceling = false;
        throw error;
      }
    },
  },
});
