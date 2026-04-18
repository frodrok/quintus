-- Create target database
CREATE DATABASE awesome_webapp;

\c awesome_webapp

CREATE OR REPLACE FUNCTION is_female(pnr TEXT) RETURNS BOOLEAN AS $$
DECLARE
  digits TEXT;
  gender_digit INT;
BEGIN
  digits := REGEXP_REPLACE(pnr, '[^0-9]', '', 'g');
  gender_digit := CAST(SUBSTRING(digits, 10, 1) AS INT);
  RETURN (gender_digit % 2) = 0;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE TABLE products (
  id SERIAL PRIMARY KEY, name TEXT NOT NULL, category TEXT NOT NULL,
  price_sek NUMERIC(10,2) NOT NULL, stock INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE customers (
  id SERIAL PRIMARY KEY, first_name TEXT NOT NULL, last_name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE, personnummer TEXT NOT NULL,
  phone TEXT, city TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
  id SERIAL PRIMARY KEY, customer_id INT NOT NULL REFERENCES customers(id),
  status TEXT NOT NULL DEFAULT 'pending', total_sek NUMERIC(10,2) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
  id SERIAL PRIMARY KEY, order_id INT NOT NULL REFERENCES orders(id),
  product_id INT NOT NULL REFERENCES products(id),
  quantity INT NOT NULL, unit_price NUMERIC(10,2) NOT NULL
);

INSERT INTO products (name, category, price_sek, stock) VALUES
  ('Laptop Pro 15','Electronics',14999.00,23),
  ('Wireless Mouse','Electronics',399.00,150),
  ('USB-C Hub','Electronics',599.00,87),
  ('Standing Desk','Furniture',4999.00,12),
  ('Ergonomic Chair','Furniture',6499.00,8),
  ('Monitor 27"','Electronics',5999.00,31),
  ('Keyboard Mechanical','Electronics',1299.00,64),
  ('Webcam HD','Electronics',899.00,45),
  ('Desk Lamp','Furniture',349.00,92),
  ('Notebook A5','Stationery',79.00,300),
  ('Pen Set','Stationery',129.00,200),
  ('Headphones','Electronics',2499.00,37),
  ('Phone Stand','Accessories',199.00,120),
  ('Cable Organizer','Accessories',149.00,180),
  ('Whiteboard','Furniture',1299.00,15),
  ('Sticky Notes','Stationery',49.00,500),
  ('External SSD','Electronics',1099.00,56),
  ('Power Bank','Electronics',699.00,78),
  ('Mouse Pad XL','Accessories',249.00,95),
  ('Laptop Sleeve','Accessories',449.00,67);

INSERT INTO customers (first_name, last_name, email, personnummer, phone, city) VALUES
  ('Erik','Johansson','erik.johansson@example.com','19850312-0131','0701234567','Stockholm'),
  ('Anna','Lindqvist','anna.lindqvist@example.com','19920615-0241','0702345678','Göteborg'),
  ('Lars','Eriksson','lars.eriksson@example.com','19780923-0131','0703456789','Malmö'),
  ('Maria','Andersson','maria.andersson@example.com','19951104-0241','0704567890','Uppsala'),
  ('Johan','Nilsson','johan.nilsson@example.com','19881219-0131','0705678901','Västerås'),
  ('Sara','Karlsson','sara.karlsson@example.com','19910307-0241','0706789012','Örebro'),
  ('Magnus','Persson','magnus.persson@example.com','19760814-0131','0707890123','Linköping'),
  ('Karin','Svensson','karin.svensson@example.com','19990501-0241','0708901234','Helsingborg'),
  ('Peter','Gustafsson','peter.gustafsson@example.com','19831127-0131','0709012345','Norrköping'),
  ('Linda','Magnusson','linda.magnusson@example.com','19940218-0241','0700123456','Jönköping'),
  ('Anders','Larsson','anders.larsson@example.com','19870609-0131','0711234567','Umeå'),
  ('Eva','Olsson','eva.olsson@example.com','19730422-0241','0712345678','Lund'),
  ('Mikael','Håkansson','mikael.hakansson@example.com','19961013-0131','0713456789','Borås'),
  ('Ingrid','Björk','ingrid.bjork@example.com','19820725-0241','0714567890','Sundsvall'),
  ('Thomas','Ström','thomas.strom@example.com','19891130-0131','0715678901','Gävle'),
  ('Annika','Lund','annika.lund@example.com','19970316-0241','0716789012','Eskilstuna'),
  ('Fredrik','Berg','fredrik.berg@example.com','19800821-0131','0717890123','Karlstad'),
  ('Helena','Åberg','helena.aberg@example.com','19930504-0241','0718901234','Täby'),
  ('Niklas','Ekström','niklas.ekstrom@example.com','19751217-0131','0719012345','Södertälje'),
  ('Cecilia','Holmgren','cecilia.holmgren@example.com','19881026-0241','0710123456','Huddinge'),
  ('David','Engström','david.engstrom@example.com','19920703-0131','0721234567','Järfälla'),
  ('Sofia','Hedlund','sofia.hedlund@example.com','19840415-0241','0722345678','Nacka'),
  ('Marcus','Söderberg','marcus.soderberg@example.com','19991108-0131','0723456789','Sollentuna'),
  ('Malin','Lindgren','malin.lindgren@example.com','19770922-0241','0724567890','Tyresö'),
  ('Andreas','Fransson','andreas.fransson@example.com','19911211-0131','0725678901','Botkyrka');

INSERT INTO orders (customer_id, status, total_sek, created_at) VALUES
  (1,'completed',15398.00,'2026-01-05 09:12:00+01'),
  (1,'completed',399.00,'2026-02-14 14:30:00+01'),
  (2,'completed',6498.00,'2026-01-18 11:05:00+01'),
  (2,'shipped',1298.00,'2026-03-02 16:45:00+01'),
  (3,'completed',5999.00,'2026-01-22 10:20:00+01'),
  (3,'completed',2498.00,'2026-02-28 13:15:00+01'),
  (4,'completed',748.00,'2026-01-30 15:00:00+01'),
  (4,'pending',4999.00,'2026-03-15 09:30:00+01'),
  (5,'completed',14999.00,'2026-02-03 12:00:00+01'),
  (5,'completed',599.00,'2026-02-20 17:20:00+01');

INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
  (1,1,1,14999.00),(1,7,1,1299.00),(2,2,1,399.00),(3,5,1,6499.00),
  (4,7,1,1299.00),(5,6,1,5999.00),(6,12,1,2499.00),(7,10,5,79.00),
  (8,4,1,4999.00),(9,1,1,14999.00),(10,3,1,599.00);