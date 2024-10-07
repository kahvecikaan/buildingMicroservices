import React from 'react';
import { Navbar, Nav, Container, Form, FormControl, Button } from 'react-bootstrap';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import CoffeeList from './CoffeeList';
import Admin from './Admin';
import './App.css';
import 'bootstrap/dist/css/bootstrap.min.css';

function App() {
    return (
        <Router>
            <div className="App">
                <Navbar bg="dark" variant="dark" expand="lg">
                    <Container>
                        <Navbar.Brand as={Link} to="/">☕ Awesome Coffee Shop</Navbar.Brand>
                        <Navbar.Toggle aria-controls="basic-navbar-nav" />
                        <Navbar.Collapse id="basic-navbar-nav">
                            <Nav className="me-auto">
                                <Nav.Link as={Link} to="/">Home</Nav.Link>
                                <Nav.Link as={Link} to="/menu">Menu</Nav.Link>
                                <Nav.Link as={Link} to="/about">About</Nav.Link>
                                <Nav.Link as={Link} to="/admin">Admin</Nav.Link>
                            </Nav>
                            <Form className="d-flex">
                                <FormControl type="text" placeholder="Search" className="me-2" />
                                <Button variant="outline-light">Search</Button>
                            </Form>
                        </Navbar.Collapse>
                    </Container>
                </Navbar>
                <Container className="mt-4">
                    <Routes>
                        <Route path="/" element={<Home />} />
                        <Route path="/menu" element={<CoffeeList />} />
                        <Route path="/about" element={<About />} />
                        <Route path="/admin" element={<Admin />} />
                    </Routes>
                </Container>
            </div>
        </Router>
    );
}

function Home() {
    return (
        <div>
            <h2>Welcome to the Awesome Coffee Shop!</h2>
            <p>Enjoy our selection of the finest coffees.</p>
        </div>
    );
}

function About() {
    return (
        <div>
            <h2>About Us</h2>
            <p>We are passionate about coffee and committed to providing the best experience.</p>
        </div>
    );
}

export default App;
