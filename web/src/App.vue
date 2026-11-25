<script setup>
import { computed, onMounted, ref } from 'vue'

const projects = ref([])
const selectedProject = ref(null)
const branches = ref([])
const tasks = ref([])
const projectBaseBranch = ref('')
const taskBaseBranch = ref('')
const view = ref('home')
const busy = ref(false)
const error = ref('')
const notice = ref('')
const createForm = ref(false)
const projectName = ref('')
const parentPath = ref('')
const existingPath = ref('')
const savingProjectBranch = ref(false)
const taskForm = ref(false)
const taskTitle = ref('')
const taskDescription = ref('')
const savingTask = ref(false)

const finalPath = computed(() => {
  if (!parentPath.value || !projectName.value.trim()) return ''
  const separator = parentPath.value.includes('\\') ? '\\' : '/'
  return parentPath.value.replace(/[\\/]+$/, '') + separator + projectName.value.trim()
})

async function api(url, options = {}) {
  const response = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  })
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.error || `Error ${response.status}`)
  return body
}

async function refreshProjects() {
  projects.value = await api('/api/projects')
}

async function chooseFolder(operation) {
  error.value = ''
  try {
    const result = await api('/api/folder-selection', {
      method: 'POST',
      body: JSON.stringify({ operation }),
    })
    if (result.cancelled) return
    if (operation === 'create') parentPath.value = result.path
    else existingPath.value = result.path
  } catch (cause) {
    error.value = `No se pudo seleccionar la carpeta. Puedes pegar la ruta en el campo de texto. (${cause.message})`
  }
}

async function createProject() {
  error.value = ''
  notice.value = ''
  busy.value = true
  try {
    const project = await api('/api/projects', {
      method: 'POST',
      body: JSON.stringify({ operation: 'create', name: projectName.value, parent_path: parentPath.value }),
    })
    await refreshProjects()
    projectName.value = ''
    parentPath.value = ''
    createForm.value = false
    openProject(project)
    notice.value = 'Proyecto creado y registrado.'
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}

async function openExisting() {
  if (!existingPath.value) return
  error.value = ''
  notice.value = ''
  busy.value = true
  try {
    const project = await api('/api/projects', {
      method: 'POST',
      body: JSON.stringify({ operation: 'open', path: existingPath.value }),
    })
    await refreshProjects()
    existingPath.value = ''
    openProject(project)
    notice.value = 'Proyecto registrado.'
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}

function openProject(project) {
  selectedProject.value = project
  projectBaseBranch.value = project.base_branch || ''
  taskBaseBranch.value = project.base_branch || ''
  branches.value = []
  tasks.value = []
  taskForm.value = false
  view.value = 'project'
  error.value = ''
  void loadProjectDetails(project.id)
}

async function loadProjectDetails(projectId) {
  try {
    const [branchResult, projectTasks] = await Promise.all([
      api(`/api/projects/${projectId}/branches`),
      api(`/api/projects/${projectId}/tasks`),
    ])
    if (selectedProject.value?.id !== projectId) return
    branches.value = branchResult.branches
    tasks.value = projectTasks
  } catch (cause) {
    error.value = cause.message
  }
}

async function saveProjectBranch() {
  if (!selectedProject.value || !projectBaseBranch.value) return
  error.value = ''
  savingProjectBranch.value = true
  try {
    const updated = await api(`/api/projects/${selectedProject.value.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ base_branch: projectBaseBranch.value }),
    })
    selectedProject.value = updated
    taskBaseBranch.value = updated.base_branch
    await refreshProjects()
    notice.value = 'Rama base del proyecto actualizada.'
  } catch (cause) {
    error.value = cause.message
  } finally {
    savingProjectBranch.value = false
  }
}

async function createTask() {
  if (!selectedProject.value || !taskTitle.value.trim() || !taskBaseBranch.value) return
  error.value = ''
  savingTask.value = true
  try {
    const task = await api(`/api/projects/${selectedProject.value.id}/tasks`, {
      method: 'POST',
      body: JSON.stringify({
        title: taskTitle.value,
        description: taskDescription.value,
        base_branch: taskBaseBranch.value,
      }),
    })
    tasks.value.unshift(task)
    taskTitle.value = ''
    taskDescription.value = ''
    taskBaseBranch.value = selectedProject.value.base_branch
    taskForm.value = false
    notice.value = `Tarea creada desde ${task.base_branch}; esa será también la rama destino de su PR.`
  } catch (cause) {
    error.value = cause.message
  } finally {
    savingTask.value = false
  }
}

function showProjects() {
  selectedProject.value = null
  view.value = 'home'
  notice.value = ''
}

onMounted(async () => {
  try {
    await refreshProjects()
  } catch (cause) {
    error.value = cause.message
  }
})
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <button class="brand" type="button" @click="showProjects">
        <span class="brand-mark">AF</span>
        <span>AI Factory</span>
      </button>
      <div class="topbar-right">
        <span class="local-indicator"><i></i> Local</span>
      </div>
    </header>

    <main class="main">
      <template v-if="view === 'home'">
        <section class="page-heading">
          <div>
            <p class="eyebrow">WORKSPACE</p>
            <h1>Proyectos</h1>
            <p class="subtitle">Elige un proyecto para continuar o añade una carpeta.</p>
          </div>
          <button class="primary-button" type="button" @click="createForm = !createForm">
            <span class="plus">＋</span> Nuevo proyecto
          </button>
        </section>

        <section v-if="createForm" class="form-card">
          <div class="form-heading">
            <div>
              <h2>Crear proyecto</h2>
              <p>AI Factory creará una carpeta nueva e inicializará Git.</p>
            </div>
            <button class="icon-button" aria-label="Cerrar" @click="createForm = false">×</button>
          </div>
          <div class="form-grid">
            <label>
              <span>Nombre del proyecto</span>
              <input v-model="projectName" autofocus placeholder="Por ejemplo, mi-aplicacion" />
            </label>
            <label>
              <span>Ubicación</span>
              <div class="path-input">
                <input v-model="parentPath" placeholder="Selecciona o pega la ruta de destino" />
                <button class="secondary-button" type="button" @click="chooseFolder('create')">Seleccionar…</button>
              </div>
            </label>
          </div>
          <p v-if="finalPath" class="path-preview">Se creará en <code>{{ finalPath }}</code></p>
          <div class="form-actions">
            <button class="text-button" type="button" @click="createForm = false">Cancelar</button>
            <button class="primary-button" :disabled="busy || !projectName.trim() || !parentPath" @click="createProject">
              {{ busy ? 'Creando…' : 'Crear proyecto' }}
            </button>
          </div>
        </section>

        <section class="open-card">
          <div class="open-copy">
            <div class="folder-icon">↗</div>
            <div>
              <h2>Abrir proyecto existente</h2>
              <p>Registra una carpeta existente que contenga un repositorio Git.</p>
            </div>
          </div>
          <div class="path-input open-control">
            <input v-model="existingPath" placeholder="Selecciona o pega la ruta de proyecto" />
            <button class="secondary-button" type="button" @click="chooseFolder('open')">Seleccionar…</button>
            <button class="primary-button" :disabled="busy || !existingPath" @click="openExisting">
              {{ busy ? 'Abriendo…' : 'Abrir' }}
            </button>
          </div>
        </section>

        <div v-if="error" class="alert error-alert" role="alert">{{ error }}</div>
        <div v-if="notice" class="alert success-alert" role="status">{{ notice }}</div>

        <section class="project-section">
          <div class="section-heading">
            <h2>Registrados</h2>
            <span class="count">{{ projects.length }}</span>
          </div>
          <div v-if="projects.length" class="project-grid">
            <button v-for="project in projects" :key="project.id" class="project-card" @click="openProject(project)">
              <span class="project-symbol">{{ project.name.slice(0, 1).toUpperCase() }}</span>
              <span class="project-info">
                <strong>{{ project.name }}</strong>
                <small>{{ project.path }}</small>
              </span>
              <span class="project-arrow">→</span>
            </button>
          </div>
          <div v-else class="empty-state">
            <div class="empty-icon">▱</div>
            <h3>Aún no hay proyectos</h3>
            <p>Crea uno nuevo o abre una carpeta que ya tengas.</p>
          </div>
        </section>
      </template>

      <template v-else>
        <button class="back-link" @click="showProjects">← Todos los proyectos</button>
        <section class="project-detail">
          <span class="project-symbol large-symbol">{{ selectedProject?.name.slice(0, 1).toUpperCase() }}</span>
          <p class="eyebrow">PROYECTO</p>
          <h1>{{ selectedProject?.name }}</h1>
          <p class="project-path">{{ selectedProject?.path }}</p>
          <div class="ready-note"><span>✓</span> El proyecto está registrado.</div>
        </section>
        <div v-if="error" class="alert error-alert" role="alert">{{ error }}</div>
        <div v-if="notice" class="alert success-alert" role="status">{{ notice }}</div>

        <section class="settings-card">
          <div>
            <h2>Rama base del proyecto</h2>
            <p>Las tareas nuevas parten de esta rama por defecto. Cada tarea puede elegir otra.</p>
          </div>
          <div class="branch-setting">
            <select v-model="projectBaseBranch" aria-label="Rama base predeterminada">
              <option v-if="projectBaseBranch && !branches.includes(projectBaseBranch)" :value="projectBaseBranch">{{ projectBaseBranch }} (configurada)</option>
              <option v-for="branch in branches" :key="branch" :value="branch">{{ branch }}</option>
            </select>
            <button class="primary-button" :disabled="savingProjectBranch || !projectBaseBranch || projectBaseBranch === selectedProject?.base_branch" @click="saveProjectBranch">
              {{ savingProjectBranch ? 'Guardando…' : 'Guardar' }}
            </button>
          </div>
        </section>

        <section class="tasks-section">
          <div class="section-heading task-heading">
            <div>
              <h2>Tareas</h2>
              <p>La rama elegida queda guardada en la tarea y define el punto de partida y destino de su PR.</p>
            </div>
            <button class="primary-button" type="button" @click="taskForm = !taskForm">＋ Nueva tarea</button>
          </div>

          <section v-if="taskForm" class="form-card task-form">
            <label>
              <span>Título</span>
              <input v-model="taskTitle" autofocus placeholder="Por ejemplo, añadir búsqueda por nombre" />
            </label>
            <label>
              <span>Descripción</span>
              <textarea v-model="taskDescription" rows="3" placeholder="Contexto y resultado esperado"></textarea>
            </label>
            <label>
              <span>Rama base de esta tarea</span>
              <select v-model="taskBaseBranch">
                <option v-if="taskBaseBranch && !branches.includes(taskBaseBranch)" :value="taskBaseBranch">{{ taskBaseBranch }} (configurada)</option>
                <option v-for="branch in branches" :key="branch" :value="branch">{{ branch }}</option>
              </select>
            </label>
            <p class="path-preview">Se creará un worktree con el contenido de <code>{{ taskBaseBranch }}</code> y la PR apuntará a esa misma rama.</p>
            <div class="form-actions">
              <button class="text-button" type="button" @click="taskForm = false">Cancelar</button>
              <button class="primary-button" :disabled="savingTask || !taskTitle.trim() || !taskBaseBranch" @click="createTask">
                {{ savingTask ? 'Creando…' : 'Crear tarea' }}
              </button>
            </div>
          </section>

          <div v-if="tasks.length" class="task-list">
            <article v-for="task in tasks" :key="task.id" class="task-card">
              <div>
                <strong>{{ task.title }}</strong>
                <p v-if="task.description">{{ task.description }}</p>
              </div>
              <div class="task-meta"><span>{{ task.status }}</span><code>{{ task.base_branch }}</code></div>
            </article>
          </div>
          <div v-else-if="!taskForm" class="empty-state task-empty">
            <h3>Aún no hay tareas</h3>
            <p>Crea una tarea y elige la rama de la que debe partir.</p>
          </div>
        </section>
      </template>
    </main>
    <footer>AI Factory <span>·</span> Tus proyectos permanecen en tu equipo</footer>
  </div>
</template>
