import React, { useState } from 'react';
import { Form, Col, Row, Button, Container } from 'react-bootstrap';
import Toast from './Toast';
import axios from 'axios';

function Admin() {
    const [validated, setValidated] = useState(false);
    const [id, setId] = useState('');
    const [file, setFile] = useState(null);
    const [buttonDisabled, setButtonDisabled] = useState(false);
    const [toastShow, setToastShow] = useState(false);
    const [toastText, setToastText] = useState('');

    const handleSubmit = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;

        if (form.checkValidity() === false) {
            event.stopPropagation();
            setValidated(true);
            return;
        }

        setButtonDisabled(true);
        setToastShow(false);

        // Create the FormData object
        const data = new FormData();
        data.append('file', file);
        data.append('id', id);

        try {
            // Upload the file
            const response = await axios.post(
                `${process.env.REACT_APP_FILES_LOCATION}/images`,
                data
                // Let Axios set 'Content-Type' automatically
            );

            if (response.status === 201) { // Updated check
                setToastText('Uploaded file successfully.');
            } else {
                setToastText(`Unable to upload file. Error: ${response.statusText}`);
            }
        } catch (error) {
            console.error('Error:', error);
            setToastText(`Unable to upload file. ${error.message}`);
        } finally {
            setButtonDisabled(false);
            setToastShow(true);
        }
    };


    const changeHandler = (event) => {
        const { name, value, files } = event.target;
        if (name === 'file') {
            setFile(files[0]);
        } else if (name === 'id') {
            setId(value);
        }
        setToastShow(false);
    };

    return (
        <div>
            <h1 style={{ marginBottom: '40px' }}>Admin</h1>
            <Container className="text-left">
                <Form noValidate validated={validated} onSubmit={handleSubmit}>
                    {/* Product ID Field */}
                    <Form.Group as={Row} controlId="productID">
                        <Form.Label column sm="2">
                            Product ID:
                        </Form.Label>
                        <Col sm="6">
                            <Form.Control
                                type="text"
                                name="id"
                                placeholder=""
                                required
                                value={id}
                                onChange={changeHandler}
                            />
                            <Form.Text className="text-muted">
                                Enter the product ID to upload an image for
                            </Form.Text>
                            <Form.Control.Feedback type="invalid">
                                Please provide a product ID.
                            </Form.Control.Feedback>
                        </Col>
                        <Col sm="4">
                            <Toast show={toastShow} message={toastText} />
                        </Col>
                    </Form.Group>

                    {/* File Upload Field */}
                    <Form.Group as={Row} controlId="fileUpload">
                        <Form.Label column sm="2">
                            File:
                        </Form.Label>
                        <Col sm="6">
                            <Form.Control
                                type="file"
                                name="file"
                                placeholder=""
                                required
                                onChange={changeHandler}
                            />
                            <Form.Text className="text-muted">
                                Image to associate with the product
                            </Form.Text>
                            <Form.Control.Feedback type="invalid">
                                Please select a file to upload.
                            </Form.Control.Feedback>
                        </Col>
                    </Form.Group>

                    {/* Submit Button */}
                    <Form.Group as={Row}>
                        <Col sm={{ span: 6, offset: 2 }}>
                            <Button type="submit" disabled={buttonDisabled}>
                                Submit form
                            </Button>
                        </Col>
                    </Form.Group>
                </Form>
            </Container>
        </div>
    );
}

export default Admin;
