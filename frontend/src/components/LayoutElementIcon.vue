<script setup lang="ts">
defineProps<{ kind: string; text?: string }>()
</script>

<template>
  <div class="el" :class="`el-${kind}`">
    <span v-if="kind === 'aisle'" class="el-text">{{ text || 'PASILLO' }}</span>
    <span v-else-if="kind === 'label'" class="el-text">{{ text }}</span>
    <span v-else-if="kind === 'icon_yape'" class="el-yape">YP</span>
    <template v-else>
      <span class="el-icon">
        <template v-if="kind === 'stairs'">⇗</template>
        <template v-else-if="kind === 'tv' || kind === 'screen'">📺</template>
        <template v-else-if="kind === 'wc'">🚻</template>
        <template v-else-if="kind === 'driver'">🚍</template>
        <template v-else-if="kind === 'seat_woman'">♀</template>
        <template v-else-if="kind === 'seat_man'">♂</template>
        <template v-else-if="kind === 'bed'">🛏</template>
        <template v-else>·</template>
      </span>
      <span v-if="text" class="el-sub">{{ text }}</span>
    </template>
  </div>
</template>

<style scoped>
.el {
  width: 100%; height: 100%;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  border-radius: 6px;
  font-size: 0.62rem; font-weight: 700; text-align: center; line-height: 1;
  min-width: 38px; min-height: 38px;
}
.el-icon { font-size: 1rem; }
.el-text { font-size: 0.6rem; padding: 0 2px; }
.el-sub { font-size: 0.52rem; color: #64748b; margin-top: 1px; font-weight: 500; }
.el-aisle { background: #ecfeff; color: #0e7490; border: 1px dashed #67e8f9; }
.el-stairs { background: #f5f3ff; color: #6d28d9; border: 1px solid #c4b5fd; }
.el-tv, .el-screen { background: #1e293b; color: white; border: 1px solid #334155; }
.el-wc { background: #ecfdf5; color: #047857; border: 1px solid #86efac; }
.el-driver { background: #fffbeb; color: #b45309; border: 1px solid #fcd34d; }
.el-seat_woman { background: #fdf2f8; color: #be185d; border: 1px solid #f9a8d4; }
.el-seat_man { background: #eff6ff; color: #1d4ed8; border: 1px solid #93c5fd; }
.el-bed { background: #fef3c7; color: #92400e; border: 1px solid #fcd34d; }
.el-label { background: transparent; color: #334155; font-weight: 800; }
.el-yape {
  background: #6b21a8; color: white; font-weight: 800; font-size: 0.7rem;
  width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; border-radius: 6px;
}
</style>
