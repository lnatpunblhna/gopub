import { createRouter, createWebHashHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import store from 'store'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'

const leftSlide = () => import('components/leftSlide/index.vue')

const routes = [{
  path: '/home',
  name: 'home',
  components: {
    default: () => import('pages/home/index.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '主页',
    auth: true
  }
}, {
  path: '/conf/list',
  name: 'confList',
  components: {
    default: () => import('pages/conf/base.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '项目配置',
    auth: true,
    admin: true
  }
}, {
  path: '/conf/update/:id',
  name: 'confUpdate',
  components: {
    default: () => import('pages/conf/save.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '修改配置',
    auth: true,
    admin: true
  }
}, {
  path: '/conf/detection/:id',
  name: 'confDetection',
  components: {
    default: () => import('pages/conf/detection.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '检测项目',
    auth: true,
    admin: true
  }
}, {
  path: '/conf/add',
  name: 'confAdd',
  components: {
    default: () => import('pages/conf/save.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '增加项目配置',
    auth: true,
    admin: true
  }
}, {
  path: '/task/list',
  name: 'taskList',
  components: {
    default: () => import('pages/task/base.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '全部上线单',
    auth: true,
    admin: true
  }
}, {
  path: '/task/create',
  name: 'taskCreate',
  components: {
    default: () => import('pages/task/create.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '创建上线单',
    auth: true
  }
}, {
  path: '/task/release/:id',
  name: 'taskRelease',
  components: {
    default: () => import('pages/task/release.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '部署上线',
    auth: true
  }
}, {
  path: '/task/mylist',
  name: 'taskMyList',
  components: {
    default: () => import('pages/task/mylist.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '我的上线单',
    auth: true
  }
}, {
  path: '/task/git',
  name: 'taskGit',
  components: {
    default: () => import('pages/task/git.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '创建上线单',
    auth: true
  }
}, {
  path: '/task/file',
  name: 'taskFile',
  components: {
    default: () => import('pages/task/file.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '创建上线单',
    auth: true
  }
}, {
  path: '/task/jenkins',
  name: 'taskJenkins',
  components: {
    default: () => import('pages/task/jenkins.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '创建上线单',
    auth: true
  }
}, {
  path: '/p2p/check',
  name: 'p2pCheck',
  components: {
    default: () => import('pages/p2p/check.vue'),
    menuView: leftSlide
  },
  meta: {
    title: 'agent状态查询',
    auth: true,
    admin: true
  }
}, {
  path: '/other/noauto',
  name: 'noauto',
  components: {
    default: () => import('pages/other/noauto.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '预发布统计',
    auth: true,
    admin: true
  }
}, {
  path: '/other/flush',
  name: 'otherFlush',
  components: {
    default: () => import('pages/task/flush.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '刷新版本号',
    auth: true,
    admin: true
  }
}, {
  path: '/other/gitpull',
  name: 'otherGitpull',
  components: {
    default: () => import('pages/task/gitpull.vue'),
    menuView: leftSlide
  },
  meta: {
    title: '预发布git版本查看',
    auth: true,
    admin: true
  }
}, {
  path: '/user/login',
  name: 'login',
  components: {
    fullView: () => import('pages/user/login.vue')
  }
}, {
  path: '/user/register',
  name: 'register',
  components: {
    menuView: leftSlide,
    default: () => import('pages/user/register.vue')
  },
  meta: {
    title: '新增用户',
    auth: true,
    admin: true
  }
}, {
  path: '/user/changepasswd',
  name: 'changepasswd',
  components: {
    menuView: leftSlide,
    default: () => import('pages/user/changepasswd.vue')
  },
  meta: {
    title: '修改密码',
    auth: true
  }
}, {
  path: '/user/list',
  name: 'userList',
  components: {
    menuView: leftSlide,
    default: () => import('pages/user/list.vue')
  },
  meta: {
    title: '用户列表',
    auth: true,
    admin: true
  }
}, {
  path: '/',
  redirect: '/home'
}, {
  path: '/:pathMatch(.*)*',
  name: 'notPage',
  components: {
    fullView: () => import('pages/error/404.vue')
  }
}]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }
    return { left: 0, top: 0 }
  }
})

// 管理员看得到「全部上线单」，普通用户只有「我的上线单」，
// 登录后的落地页与登录态跳转都按这个区分
function home_path(is_admin) {
  return is_admin ? '/task/list' : '/task/mylist'
}

router.beforeEach((to, from, next) => {
  NProgress.start()
  const toName = to.name
  const isLogin = store.state.user_info.login
  const user = store.state.user_info.user
  const isAdmin = Number(user && user.Role) === 1

  if (!isLogin && toName !== 'login') {
    next({ name: 'login' })
  } else if (isLogin && toName === 'login') {
    next({ path: home_path(isAdmin) })
  } else if (isLogin && to.meta && to.meta.admin && !isAdmin) {
    // 菜单已按 adminOnly 隐藏了这些入口，这里再拦一道：
    // 光隐藏菜单挡不住直接在地址栏敲 URL
    ElMessage({ message: '无权访问该页面', type: 'warning' })
    next({ path: home_path(isAdmin) })
  } else {
    next()
  }
})

router.afterEach(() => {
  NProgress.done()
})

export default router
