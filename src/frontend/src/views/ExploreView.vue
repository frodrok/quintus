<template>
  <v-container fluid class="pa-0 fill-height">
    <v-row no-gutters class="fill-height">

      <!-- Schema browser -->
      <v-col cols="2" class="panel-col">
        <div class="panel-inner">
          <div class="panel-title">Schema</div>
          <div class="schema-scroll">
            <div v-if="!selectedConnectionId" class="empty-hint">Select a connection</div>
            <div v-else-if="schemaLoading" class="empty-hint">Loading…</div>
            <div v-else-if="!schema.length" class="empty-hint">No tables found</div>
            <div v-else>
              <div v-for="(tables, schemaName) in schemaTree" :key="schemaName" class="schema-group">
                <div class="schema-name-row" @click="toggleSchema(String(schemaName))">
                  <v-icon size="14" class="mr-1">{{ schemaOpen[String(schemaName)] ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>
                  <span>{{ schemaName }}</span>
                </div>
                <div v-if="schemaOpen[String(schemaName)]">
                  <div v-for="t in tables" :key="t.table">
                    <div class="table-name-row" @click="insertTable(t.schema, t.table)" :title="'Insert ' + t.table">
                      <v-icon size="12" class="mr-1" :color="t.table.startsWith('ƒ') ? 'purple' : 'primary'">
                        {{ t.table.startsWith('ƒ') ? 'mdi-function' : 'mdi-table' }}
                      </v-icon>
                      <span>{{ t.table }}</span>
                    </div>
                    <div class="col-list">
                      <div v-for="col in t.columns" :key="col.name" class="col-row">
                        <span class="col-name">{{ col.name }}</span>
                        <span class="col-type">{{ col.data_type }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </v-col>

      <!-- Editor + results -->
      <v-col cols="10" class="panel-col" style="border-left: 1px solid rgba(0,0,0,0.08)">
        <div class="panel-inner">

          <!-- Toolbar -->
          <div class="explore-toolbar">
            <v-select
              v-model="selectedConnectionId"
              :items="connections"
              item-title="name"
              item-value="id"
              placeholder="Select connection"
              variant="outlined"
              density="compact"
              hide-details
              style="max-width: 220px"
            />
            <v-btn
              color="primary"
              size="small"
              :loading="running"
              :disabled="!selectedConnectionId || !sql.trim()"
              @click="run"
              prepend-icon="mdi-play"
            >
              Run
            </v-btn>
            <span class="text-caption text-medium-emphasis">Ctrl+Enter to run</span>
            <v-spacer />
            <div v-if="result" class="d-flex gap-1">
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadCSV(result!, 'export.csv')">CSV</v-btn>
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadExcel(result!, 'export.xlsx')">Excel</v-btn>
            </div>
          </div>

          <!-- SQL editor -->
          <div class="sql-area">
            <textarea
              v-model="sql"
              class="sql-input"
              spellcheck="false"
              placeholder="SELECT * FROM customers LIMIT 100"
              @keydown.ctrl.enter.prevent="run"
              @keydown.meta.enter.prevent="run"
            />
          </div>

          <!-- Results toolbar -->
          <div class="results-toolbar">
            <template v-if="result">
              <v-chip v-if="result.masked_columns?.length" size="x-small" color="warning" class="mr-2">
                {{ result.masked_columns.length }} masked
              </v-chip>
              <span class="text-body-2">
                {{ result.row_count }} row{{ result.row_count !== 1 ? 's' : '' }}
                <v-chip v-if="result.truncated" size="x-small" color="warning" class="mx-1">truncated</v-chip>
                <span class="text-medium-emphasis text-caption"> · {{ result.run_id.slice(0, 8) }}</span>
              </span>
            </template>
            <span v-else-if="runError" class="text-error text-caption">{{ runError }}</span>
            <span v-else class="text-medium-emphasis text-caption">No results yet</span>
          </div>

          <!-- Results table -->
          <div class="results-scroll">
            <v-table v-if="result" density="compact" fixed-header>
              <thead>
                <tr>
                  <th v-for="col in result.columns" :key="col.name" class="text-left">
                    {{ col.name }}
                    <span class="text-caption text-medium-emphasis ml-1">{{ col.type }}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, ri) in result.rows" :key="ri">
                  <td v-for="(val, ci) in row" :key="ci" class="text-caption">{{ formatValue(val) }}</td>
                </tr>
              </tbody>
            </v-table>
            <div v-else-if="running" class="centered-hint">
              <v-progress-circular indeterminate size="24" class="mr-2" />Running…
            </div>
            <div v-else class="centered-hint text-medium-emphasis">
              Select a connection, write SQL and press Run
            </div>
          </div>

        </div>
      </v-col>

    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { listConnections, getConnectionSchema, type Connection, type SchemaTable } from '../api/connections'
import { createRun, type RunResult, downloadCSV, downloadExcel } from '../api/runs'

const connections = ref<Connection[]>([])
const selectedConnectionId = ref<string>('')
const sql = ref('')
const result = ref<RunResult | null>(null)
const runError = ref('')
const running = ref(false)

const schema = ref<SchemaTable[]>([])
const schemaOpen = ref<Record<string, boolean>>({})
const schemaLoading = ref(false)

const schemaTree = computed(() => {
  const tree: Record<string, SchemaTable[]> = {}
  for (const t of schema.value) {
    if (!tree[t.schema]) tree[t.schema] = []
    tree[t.schema].push(t)
  }
  return tree
})

watch(selectedConnectionId, async (id) => {
  schema.value = []
  schemaOpen.value = {}
  result.value = null
  runError.value = ''
  if (!id) return
  schemaLoading.value = true
  try {
    schema.value = await getConnectionSchema(id)
    const first = schema.value[0]?.schema
    if (first) schemaOpen.value = { [first]: true }
  } catch {
  } finally {
    schemaLoading.value = false
  }
})

function toggleSchema(name: string) {
  schemaOpen.value[name] = !schemaOpen.value[name]
}

function insertTable(schemaName: string, table: string) {
  const cleaned = table.startsWith('ƒ ') ? table.slice(2) : table
  const insert = schemaName === 'public' ? cleaned : `${schemaName}.${cleaned}`
  sql.value = sql.value ? sql.value + '\n' + insert : insert
}

async function run() {
  if (!selectedConnectionId.value || !sql.value.trim()) return
  running.value = true
  runError.value = ''
  result.value = null
  try {
    result.value = await createRun({
      connection_id: selectedConnectionId.value,
      sql: sql.value,
    })
  } catch (e: any) {
    runError.value = e.response?.data?.error ?? e.message
  } finally {
    running.value = false
  }
}

function formatValue(val: any): string {
  if (val === null || val === undefined) return 'NULL'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

onMounted(async () => {
  connections.value = await listConnections()
  if (connections.value.length === 1) {
    selectedConnectionId.value = connections.value[0].id
  }
})
</script>

<style scoped>
.panel-col {
  height: calc(100vh - 64px);
  overflow: hidden;
}

.panel-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #888;
  padding: 10px 12px 8px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
}

/* Schema */
.schema-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.schema-group { margin-bottom: 2px; }

.schema-name-row {
  display: flex;
  align-items: center;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  color: #444;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.schema-name-row:hover { background: #f3f4f6; border-radius: 4px; }

.table-name-row {
  display: flex;
  align-items: center;
  padding: 3px 10px 3px 20px;
  font-size: 12px;
  color: #2563eb;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.table-name-row:hover { background: #eff6ff; border-radius: 4px; }

.col-list { padding-left: 28px; }
.col-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1px 8px;
  font-size: 11px;
}
.col-name { color: #555; }
.col-type {
  color: #aaa;
  background: #f3f4f6;
  border-radius: 2px;
  padding: 0 3px;
  font-size: 10px;
  flex-shrink: 0;
  margin-left: 4px;
}

/* Toolbar */
.explore-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
}

/* SQL */
.sql-area {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
}

.sql-input {
  width: 100%;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  resize: vertical;
  line-height: 1.6;
  min-height: 140px;
  max-height: 300px;
  background: #fafafa;
  color: #1a1a1a;
  box-sizing: border-box;
}
.sql-input:focus {
  outline: none;
  border-color: #2563eb;
  background: #fff;
}

/* Results */
.results-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
  min-height: 38px;
}

.results-scroll {
  flex: 1;
  overflow: auto;
}

.centered-hint {
  height: 100%;
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #aaa;
  font-size: 14px;
}

.empty-hint {
  padding: 12px;
  font-size: 12px;
  color: #aaa;
}
</style>