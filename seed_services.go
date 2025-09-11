package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Service represents the service structure
type Service struct {
	Name           string
	NameAr         string
	Description    string
	DescriptionAr  string
	Category       string
	ImageURL       string
	BasePrice      float64
	PriceUnit      string
	Duration       string
	Guarantee      string
	Policies       string
}

func j() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Database connection parameters
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "repair_service_db")
	dbSSLMode := getEnv("DB_SSL_MODE", "disable")

	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("✅ Successfully connected to database")

	// Check if services already exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM services").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check services count:", err)
	}

	if count > 0 {
		log.Printf("⚠️  Services already exist (%d services found). Skipping insertion.", count)
		return
	}

	// Define services to insert
	services := []Service{
		{
			Name:          "Plomberie",
			NameAr:        "سباكة",
			Description:   "Services de plomberie professionnels incluant réparation de fuites, installation de robinets, réparation de chauffe-eau et maintenance des systèmes d'égout. Intervention rapide et garantie qualité.",
			DescriptionAr: "خدمات سباكة احترافية تشمل إصلاح التسريبات وتركيب الحنفيات وإصلاح سخانات المياه وصيانة أنظمة الصرف الصحي. تدخل سريع وضمان الجودة",
			Category:      "Plomberie",
			ImageURL:      "https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&h=600&fit=crop&crop=center",
			BasePrice:     1500.00,
			PriceUnit:     "par heure",
			Duration:      "1-3 heures",
			Guarantee:     "6 mois",
			Policies:      "Intervention sous 24h, devis gratuit, garantie pièces et main d'œuvre, paiement sécurisé après validation du travail, nettoyage du chantier inclus.",
		},
		{
			Name:          "Électricité",
			NameAr:        "كهرباء",
			Description:   "Services électriques complets : installation électrique, réparation de panneaux, éclairage LED, sécurité électrique et maintenance préventive. Techniciens certifiés.",
			DescriptionAr: "خدمات كهربائية شاملة: التركيب الكهربائي وإصلاح لوحات التوزيع والإضاءة LED والأمان الكهربائي والصيانة الوقائية. فنيون معتمدون",
			Category:      "Électricité",
			ImageURL:      "https://images.unsplash.com/photo-1621905251189-08b45d6a269e?w=800&h=600&fit=crop&crop=center",
			BasePrice:     2000.00,
			PriceUnit:     "par heure",
			Duration:      "2-4 heures",
			Guarantee:     "12 mois",
			Policies:      "Techniciens certifiés, diagnostic gratuit, respect des normes de sécurité, garantie complète sur tous travaux, matériaux de qualité supérieure.",
		},
		{
			Name:          "Peinture",
			NameAr:        "دهان",
			Description:   "Services de peinture intérieure et extérieure, préparation des surfaces, finitions décoratives et rénovation complète des murs et plafonds. Peintures écologiques disponibles.",
			DescriptionAr: "خدمات الدهان الداخلي والخارجي وإعداد الأسطح والطلاءات الزخرفية والتجديد الكامل للجدران والأسقف. دهانات صديقة للبيئة متوفرة",
			Category:      "Peinture",
			ImageURL:      "https://images.unsplash.com/photo-1589939705384-5185137a7f0f?w=800&h=600&fit=crop&crop=center",
			BasePrice:     800.00,
			PriceUnit:     "par m²",
			Duration:      "1-3 jours",
			Guarantee:     "3 ans",
			Policies:      "Peintures écologiques disponibles, nettoyage inclus, protection des meubles, satisfaction garantie, finitions soignées, couleurs personnalisées.",
		},
		{
			Name:          "Climatisation",
			NameAr:        "تكييف",
			Description:   "Installation, réparation et maintenance de systèmes de climatisation et chauffage, nettoyage des filtres et optimisation énergétique. Service d'urgence 24/7 disponible.",
			DescriptionAr: "تركيب وإصلاح وصيانة أنظمة التكييف والتدفئة وتنظيف المرشحات وتحسين كفاءة الطاقة. خدمة طوارئ متوفرة على مدار الساعة",
			Category:      "Climatisation",
			ImageURL:      "https://images.unsplash.com/photo-1581578731548-c64695cc6952?w=800&h=600&fit=crop&crop=center",
			BasePrice:     3000.00,
			PriceUnit:     "par intervention",
			Duration:      "2-6 heures",
			Guarantee:     "18 mois",
			Policies:      "Service d'urgence 24/7, pièces d'origine, maintenance préventive incluse, économies d'énergie garanties, diagnostic complet gratuit.",
		},
		{
			Name:          "Menuiserie",
			NameAr:        "نجارة",
			Description:   "Fabrication et réparation de meubles sur mesure, portes, fenêtres, escaliers et aménagements intérieurs en bois de qualité. Design personnalisé et finitions soignées.",
			DescriptionAr: "تصنيع وإصلاح الأثاث المخصص والأبواب والنوافذ والسلالم والتجهيزات الداخلية من الخشب عالي الجودة. تصميم مخصص وطلاءات دقيقة",
			Category:      "Menuiserie",
			ImageURL:      "https://images.unsplash.com/photo-1504148455328-c376907d081c?w=800&h=600&fit=crop&crop=center",
			BasePrice:     2500.00,
			PriceUnit:     "par projet",
			Duration:      "3-7 jours",
			Guarantee:     "5 ans",
			Policies:      "Bois certifiés, design personnalisé, finitions soignées, garantie structurelle longue durée, conseils d'entretien inclus.",
		},
		{
			Name:          "Maçonnerie",
			NameAr:        "بناء",
			Description:   "Construction et rénovation de murs, fondations, terrasses, allées et structures en béton avec matériaux durables. Respect des normes de construction.",
			DescriptionAr: "بناء وتجديد الجدران والأساسات والتراسات والممرات والهياكل الخرسانية باستخدام مواد متينة. احترام معايير البناء",
			Category:      "Maçonnerie",
			ImageURL:      "https://images.unsplash.com/photo-1541888946425-d81bb19240f5?w=800&h=600&fit=crop&crop=center",
			BasePrice:     1200.00,
			PriceUnit:     "par m²",
			Duration:      "2-5 jours",
			Guarantee:     "10 ans",
			Policies:      "Matériaux certifiés, respect des normes de construction, suivi de chantier, garantie structurelle décennale, nettoyage complet.",
		},
		{
			Name:          "Nettoyage",
			NameAr:        "تنظيف",
			Description:   "Services de nettoyage professionnel : nettoyage résidentiel, commercial, après rénovation et entretien régulier des locaux. Produits écologiques et personnel formé.",
			DescriptionAr: "خدمات التنظيف الاحترافية: التنظيف السكني والتجاري وبعد التجديد والصيانة الدورية للمباني. منتجات صديقة للبيئة وموظفون مدربون",
			Category:      "Nettoyage",
			ImageURL:      "https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&h=600&fit=crop&crop=center",
			BasePrice:     500.00,
			PriceUnit:     "par heure",
			Duration:      "2-8 heures",
			Guarantee:     "Satisfaction",
			Policies:      "Produits écologiques, personnel formé, satisfaction garantie, nettoyage en profondeur inclus, matériel professionnel, horaires flexibles.",
		},
		{
			Name:          "Jardinage",
			NameAr:        "بستنة",
			Description:   "Entretien des jardins, aménagement paysager, taille des arbres, irrigation automatique et création d'espaces verts. Plantes adaptées au climat local.",
			DescriptionAr: "صيانة الحدائق وتنسيق المناظر الطبيعية وتقليم الأشجار والري التلقائي وإنشاء المساحات الخضراء. نباتات متكيفة مع المناخ المحلي",
			Category:      "Jardinage",
			ImageURL:      "https://images.unsplash.com/photo-1585320806297-9794b3e4eeae?w=800&h=600&fit=crop&crop=center",
			BasePrice:     800.00,
			PriceUnit:     "par heure",
			Duration:      "2-6 heures",
			Guarantee:     "Saison",
			Policies:      "Plantes adaptées au climat, entretien saisonnier, conseils personnalisés, garantie de reprise, irrigation intelligente, design paysager.",
		},
		{
			Name:          "Serrurerie",
			NameAr:        "قفل",
			Description:   "Ouverture de portes, changement de serrures, installation de systèmes de sécurité, clés de secours et dépannage d'urgence. Intervention rapide 24/7.",
			DescriptionAr: "فتح الأبواب وتغيير الأقفال وتركيب أنظمة الأمان والمفاتيح الاحتياطية وإصلاح الطوارئ. تدخل سريع على مدار الساعة",
			Category:      "Serrurerie",
			ImageURL:      "https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&h=600&fit=crop&crop=center",
			BasePrice:     2500.00,
			PriceUnit:     "par intervention",
			Duration:      "30 min - 2h",
			Guarantee:     "12 mois",
			Policies:      "Intervention d'urgence 24/7, serrures certifiées, sécurité renforcée, garantie sur pièces et main d'œuvre, clés de secours.",
		},
		{
			Name:          "Vitrerie",
			NameAr:        "زجاج",
			Description:   "Remplacement de vitres cassées, installation de vitres isolantes, miroirs, vitrines commerciales et réparation d'urgence. Vitres de sécurité et isolation thermique.",
			DescriptionAr: "استبدال الزجاج المكسور وتركيب الزجاج العازل والمرايا والواجهات التجارية والإصلاح الطارئ. زجاج أمان وعزل حراري",
			Category:      "Vitrerie",
			ImageURL:      "https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&h=600&fit=crop&crop=center",
			BasePrice:     1800.00,
			PriceUnit:     "par m²",
			Duration:      "1-4 heures",
			Guarantee:     "6 mois",
			Policies:      "Vitres de sécurité, isolation thermique, intervention rapide, garantie anti-casse, nettoyage professionnel, finitions soignées.",
		},
	}

	// Insert services
	log.Println("🚀 Starting to insert services...")
	
	insertQuery := `
		INSERT INTO services (
			name, name_ar, description, description_ar, category, 
			image_url, price, duration, is_active, created_at, updated_at,
			base_price, price_unit, guarantee, policies, deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	now := time.Now()
	insertedCount := 0

	for _, service := range services {
		_, err := db.Exec(insertQuery,
			service.Name,
			service.NameAr,
			service.Description,
			service.DescriptionAr,
			service.Category,
			service.ImageURL,
			service.BasePrice, // price column
			service.Duration,
			true, // is_active
			now,  // created_at
			now,  // updated_at
			service.BasePrice, // base_price column
			service.PriceUnit,
			service.Guarantee,
			service.Policies,
			nil,  // deleted_at (NULL for active services)
		)

		if err != nil {
			log.Printf("❌ Failed to insert service '%s': %v", service.Name, err)
		} else {
			log.Printf("✅ Successfully inserted: %s (%s)", service.Name, service.Category)
			insertedCount++
		}
	}

	log.Printf("🎉 Insertion completed! %d out of %d services inserted successfully", insertedCount, len(services))

	// Verify the insertion
	log.Println("🔍 Verifying inserted services...")
	rows, err := db.Query("SELECT id, name, name_ar, category, base_price, price_unit, duration, guarantee, is_active FROM services ORDER BY id")
	if err != nil {
		log.Fatal("Failed to query services:", err)
	}
	defer rows.Close()

	log.Println("📋 Inserted Services:")
	log.Println("ID | Name | Arabic Name | Category | Price | Unit | Duration | Guarantee | Active")
	log.Println("---|------|-------------|----------|-------|------|----------|-----------|-------")

	for rows.Next() {
		var id int
		var name, nameAr, category, priceUnit, duration, guarantee string
		var basePrice float64
		var isActive bool

		err := rows.Scan(&id, &name, &nameAr, &category, &basePrice, &priceUnit, &duration, &guarantee, &isActive)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		log.Printf("%d | %s | %s | %s | %.0f | %s | %s | %s | %t", 
			id, name, nameAr, category, basePrice, priceUnit, duration, guarantee, isActive)
	}

	if err = rows.Err(); err != nil {
		log.Fatal("Error iterating over rows:", err)
	}

	log.Println("✨ Service seeding completed successfully!")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
