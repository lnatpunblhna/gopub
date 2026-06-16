<template>
    <div class="left-side">
        <div class="left-side-inner">
            <router-link to="/" class="logo block">
                <div>gopub</div>
            </router-link>
            <el-menu
                    class="menu-box"
                    router
                    :default-active="$route.path">
                <el-menu-item
                        class="menu-list"
                        v-for="item in nav_menu_data"
                        :key="item.path"
                        :index="item.path">
                    <i class="icon fa" :class="item.icon"></i>
                    <span v-text="item.title" class="text"></span>
                </el-menu-item>
            </el-menu>
        </div>
    </div>
</template>
<script type="text/javascript">
    export default{
        data(){
            return {
                nav_menu_data: [{
                    title: "登录",
                    path: "/user/login",
                    icon: "fa-sign-in"
                }]
            }
        }
    }
</script>
