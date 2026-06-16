<template>
    <footer class="footer" v-once>
        Copyright &copy; {{getYear}}
        <a href="https://github.com/lnatpunblhna" target="_blank" rel="noopener noreferrer">lnatpunblhna</a>
    </footer>
</template>
<script type="text/javascript">
    export default {
        computed: {
            getYear(){
                return new Date().getFullYear()
            }
        }
    }
</script>
