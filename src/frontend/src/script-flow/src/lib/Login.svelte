<script lang="ts">
    import { currentUser, pb } from './pocketbase';
    let username: string = '';
    let password: string = '';

    async function signIn() {
        await pb.collection('users').authWithPassword(username, password);
    }

    // async function signUp() {
    //     const data = {
    //         username, 
    //         password,
    //         confirmPassword: password,
    //         name: 'John Doe',
    //     };
    //     try {
    //         const newUser = await pb.collection('users').create(data);
    //         await signIn();
    //     } catch (error) {
    //         console.error(error);
    //     }
    //     await signIn();
    // }

    function signOut() {
        pb.authStore.clear();
    }

</script>

{#if $currentUser}
    <h1>Script flow</h1>
    <p>Welcome, {$currentUser.username}!</p>
    <button on:click={signOut}>Logout</button>
{:else}
    <h1>Script flow</h1>
    <p>Please log in.</p>
    <form on:submit|preventDefault>
        <input type="text" bind:value={username} placeholder="Username" />
        <input type="password" bind:value={password} placeholder="Password" />
    </form>
    <button on:click={signIn}>Login</button>
{/if}