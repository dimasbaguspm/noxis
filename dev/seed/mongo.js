// The demo documents: one collection with a couple of shapes, so the explorer has something to open.
const shop = db.getSiblingDB("demo");
shop.customers.insertMany(
  [
    { email: "ada@example.com", name: "Ada Lovelace", tier: "gold", orders: 3 },
    { email: "grace@example.com", name: "Grace Hopper", tier: "silver", orders: 1 },
    { email: "alan@example.com", name: "Alan Turing", tier: "gold", orders: 0 },
  ],
  { ordered: false },
);
print("seeded demo.customers: " + shop.customers.countDocuments({}));
