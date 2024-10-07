import React, { useState, useEffect } from 'react';
import { Toast as BootstrapToast } from 'react-bootstrap';

function Toast({ show, message }) {
    const [visible, setVisible] = useState(show);

    useEffect(() => {
        setVisible(show);
    }, [show]);

    const hide = () => {
        setVisible(false);
    };

    return (
        <BootstrapToast onClose={hide} show={visible} delay={3000} autohide>
            <BootstrapToast.Header>
                <strong className="me-auto">File Upload</strong>
            </BootstrapToast.Header>
            <BootstrapToast.Body>{message}</BootstrapToast.Body>
        </BootstrapToast>
    );
}

export default Toast;
