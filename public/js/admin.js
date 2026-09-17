document.addEventListener('DOMContentLoaded', function() {
    // Auto cerrar alerts
    document.querySelectorAll('.alert').forEach(alert => {
        setTimeout(() => {
            try {
                const bsAlert = new bootstrap.Alert(alert);
                bsAlert.close();
            } catch (e) {}
        }, 5000);
    });
});
