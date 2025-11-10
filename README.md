# Problem

Belum ada Open Source Scheduler buatan indonesia yang sangat customable

Existing : 

**Open Cluster** Management (OCM), Karmada di khususkan untuk kubernetes, belum untuk solusi umum

Volcano di khususkan untuk Ai

**HashiCorp Nomad:** yang kita bangun semacam ini , tapi kita terbuka tidak terbatas jenis engine  container terntentu

# Approach

Pattern Control Plane x Data Plane

# Scope [Demo Version]

- Simple Scheduler
- Provisioning with incus
- **Persistent storage using etcd** - See [ETCD_SETUP.md](README/ETCD_SETUP.md) for details
- **Cluster-aware job scheduling** - See [CLUSTER_TERMINOLOGY.md](README/CLUSTER_TERMINOLOGY.md) for details

## 📚 Documentation

All documentation has been organized in the [`README/`](README/) folder:

- **[Documentation Index](README/INDEX.md)** - Complete guide to all documentation
- **Setup Guides**: [etcd](README/ETCD_SETUP.md), [gRPC](README/GRPC_SETUP.md)
- **Architecture**: [Centro](README/CENTRO.md), [Agent](README/AGENT.md), [Scheduler](README/SCHEDULER.md)
- **API Documentation**: [REST API](README/REST_API_IMPLEMENTATION.md), [Swagger](README/SWAGGER.md)
- **Features**: [Cluster Scheduling](README/CLUSTER_TERMINOLOGY.md), [Workflows](README/workflow.md)

# Control Plane x Data Plane

## Analogi

Ini adalah salah satu konsep terpenting dalam jaringan modern. Cara termudah untuk memikirkannya adalah menggunakan **analogi Ketua Partai Politik dan Anggotanya**.

- **Control Plane (Si "Ketua Partai" / Petinggi Partai):** Ini adalah pusat strategi partai.
    - Tugasnya adalah "berpikir", "merumuskan strategi", dan "memutuskan arahan".
    - Ia mengumpulkan informasi (data survei, masukan dari daerah, isu politik, tujuan partai).
    - Ia menghitung *strategi terbaik* untuk menang (misalnya, menentukan isu kampanye, daerah prioritas).
    - Ia **tidak** benar-benar turun ke setiap TPS atau mengetuk setiap pintu rumah pemilih.
- **Data Plane (Si "Anggota / Kader Partai"):** Ini adalah para kader yang bekerja di lapangan.
    - Tugasnya adalah "bertindak" dan "melaksanakan" arahan.
    - Ia mengambil instruksi spesifik dari pusat (misalnya, "Sebarkan poster A," "Fokus pada isu B," "Kunjungi wilayah C").
    - Ia *secara fisik menemui pemilih* (data/paket) dan menyampaikan pesan partai untuk mencapai tujuan.
    - Ia bekerja serentak, cepat, dan masif di banyak tempat, tetapi hanya melakukan apa yang diinstruksikan oleh si "ketua".

### Mengapa Memisahkan Keduanya?

- **Skalabilitas:** Anda bisa memiliki satu "otak" / "Pusat" (Control Plane) yang kuat yang mengelola *banyak* "kader di lapangan" (Data Plane) yang sederhana. Ini adalah ide inti dari **Software-Defined Networking (SDN)** dan juga Kubernetes.
- **Ketahanan (Resilience):** Jika "Pusat" (Control Plane Kubernetes) *crash*, para "Kader" (Data Plane ) akan terus menjalankan aplikasi (*pods*) yang sudah ada berdasarkan arahan terakhir. Aplikasi Anda tetap berjalan.
- **Fleksibilitas:** Anda dapat memperbarui "strategi partai" (misalnya, meng-upgrade Kubernetes Control Plane) tanpa harus menghentikan dan melatih ulang semua kader (me-reboot semua *worker node*) secara bersamaan.
- **Manajemen Lingkungan Dinamis:** Seperti dalam contoh Kubernetes, "kader" (Data Plane, atau *pods* dan *nodes*) bisa muncul dan hilang dengan cepat. dan bisa jadi kader (Node) bisa berubah alamat (IP). "Pusat" (Control Plane) tidak perlu melacak setiap perubahan secara manual. Sebaliknya, setiap "Kader" baru (Node) secara proaktif mendaftarkan dirinya ke "Pusat" , sehingga klaster dapat mengelola dirinya sendiri secara dinamis.
- Control Plane (pusat), tidak perlu hit ke semua node (data plane), untuk memastikan kesehatan
    
    Mengunakan konsep heartbeat , data plane yang akan mengabarkan kesehatan ke pusat

    ```mermaid
    sequenceDiagram
        participant Agent as Agent (Data Plane)
        participant Centro as Centro (Control Plane)
        participant etcd as etcd (Persistent Storage)

        Agent->>Centro: Register node (heartbeat, metadata)
        Centro->>etcd: Save node info
        Agent->>Centro: Get job (request)
        Centro->>etcd: Check job queue & assign job
        Centro-->>Agent: Return job description
        Agent->>Agent: Run assigned job (container, etc)
        Agent->>Centro: Update status (running, completed, failed)
        Centro->>etcd: Store job status update
        Note right of Agent: Agent repeats heartbeat and job check periodically

        Centro-->>etcd: Periodic state sync\naudit, reconciliation
    ```
