package main

import (
	"log"
	"repair-service-server/database"
	"repair-service-server/models"
)

func seedServiceCategories() error {
	db := database.GetDB()

	categories := []models.ServiceCategory{
		{
			Name:        "Nettoyage à la demande",
			Description: "Service de nettoyage professionnel à domicile ou au bureau",
			Icon:        "sparkles",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   1,
		},
		{
			Name:        "Abonnement nettoyage",
			Description: "Nettoyage régulier de votre maison ou entreprise sur abonnement",
			Icon:        "refresh",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       true,
			SortOrder:   2,
		},
		{
			Name:        "Plomberie",
			Description: "Réparation de fuites, robinets et installations de plomberie",
			Icon:        "water",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   3,
		},
		{
			Name:        "Électricité",
			Description: "Installation et réparation électrique, y compris panneaux solaires",
			Icon:        "flash",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       true,
			SortOrder:   4,
		},
		{
			Name:        "Climatisation",
			Description: "Installation et entretien de climatiseurs et ventilation",
			Icon:        "snow",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   5,
		},
		{
			Name:        "Peinture",
			Description: "Peinture intérieure et extérieure, préparation et finitions",
			Icon:        "paint-roller",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   6,
		},
		{
			Name:        "Chauffe-eau",
			Description: "Installation et réparation de chauffe-eau et systèmes solaires",
			Icon:        "thermometer",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   7,
		},
		{
			Name:        "Menuiserie & Serrurerie",
			Description: "Réparation de portes/fenêtres, meubles et serrures",
			Icon:        "key",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   8,
		},
		{
			Name:        "Appareils électroménagers",
			Description: "Réparation de frigos, machines à laver et autres appareils",
			Icon:        "tools",
			Color:       "#eb5436",
			IsActive:    true,
			IsNew:       false,
			SortOrder:   9,
		},
	}

	for _, category := range categories {
		var existingCategory models.ServiceCategory
		if err := db.Where("name = ?", category.Name).First(&existingCategory).Error; err != nil {
			// Category doesn't exist, create it
			if err := db.Create(&category).Error; err != nil {
				log.Printf("Failed to create category %s: %v", category.Name, err)
				return err
			}
			log.Printf("✅ Created category: %s", category.Name)
		} else {
			log.Printf("⏭️  Category already exists: %s", category.Name)
		}
	}

	return nil
}