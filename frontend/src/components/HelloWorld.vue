<script lang="ts" setup>
import { reactive } from "vue";
import { Greet } from "../../wailsjs/go/backend/App";

const data = reactive({
  name: "",
  resultText: "Please enter your name below 👇",
});

function greet() {
  Greet(data.name).then((result: string) => {
    data.resultText = result;
  });
}
</script>

<template>
  <div class="p-8">
    <div class="max-w-2xl mx-auto space-y-6">
      <div class="card bg-base-200">
        <div class="card-body">
          <h2 class="card-title">Hello World</h2>
          <div id="result" class="result">{{ data.resultText }}</div>
          <div id="input" class="input-box flex gap-2">
            <input
              id="name"
              v-model="data.name"
              autocomplete="off"
              class="input input-bordered flex-1"
              type="text"
              placeholder="Enter your name"
            />
            <button class="btn btn-primary" @click="greet">Greet</button>
          </div>
        </div>
      </div>

      <div class="card bg-base-200">
        <div class="card-body">
          <h2 class="card-title">DaisyUI Components</h2>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-primary">Primary</button>
            <button class="btn btn-secondary">Secondary</button>
            <button class="btn btn-accent">Accent</button>
            <button class="btn btn-info">Info</button>
            <button class="btn btn-success">Success</button>
            <button class="btn btn-warning">Warning</button>
            <button class="btn btn-error">Error</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
