# 📘 Using Topics in Bindu

Bindu uses a simple and powerful topic format:

```
$type/$group/$producer
```

This structure makes it easy to understand where data is coming from, how to subscribe to it, and how to organize devices.

---

## 🧠 What is a Topic?

A topic is the channel through which messages are sent and received.

### Example:

```
InventoryUpdate/store3/deviceA
```

This means:

- `InventoryUpdate`: the type of message (inventory update)
- `store3`: the group or store
- `deviceA`: the device or system that sent it

---

## 📥 How to Subscribe

Depending on what you want to monitor, you can use wildcard filters:

- All inventory messages:

  ```
  InventoryUpdate/#
  ```

- All messages from your store:

  ```
  InventoryUpdate/store3/#
  ```

- Messages from one device:

  ```
  InventoryUpdate/store3/deviceA
  ```

---

## 📤 How to Publish

You **don’t** publish directly to topics. Instead, all messages are sent to:

```
bindu/in
```

The system will read the content of the message and forward it to the correct topic automatically.

---

## ⚠️ Error Topics

If there is a problem with your message (e.g. it’s missing a field or has invalid data), the system will send an error message to:

```
ErrorEvent/<your-group>/<your-device>
```

You can subscribe to this topic to get feedback about what went wrong.

---

## ✅ Summary

Bindu topics are:

- Easy to read
- Easy to subscribe to
- Safe and consistent

Use the format:

```
$type/$group/$producer
```

...and you're ready to work with Bindu!

