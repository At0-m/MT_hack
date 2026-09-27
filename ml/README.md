Отчет написан в `report.ipynb`

Там находятся "Данные, внешние источники и область применимости
модели"

По структуре:
```
.
├── onnx-models - модели в формате ONNX
│   ├── catboost
│   │   ├── route_11_seed239.cbm
│   │   ├── route_11_seed239.onnx
- - -- - - - - - - - - -- - - - 
│   │   ├── route_7_seed43.cbm
│   │   └── route_7_seed43.onnx
│   └── lightgbm
│       ├── lgbm_seed_239.onnx
│       ├── lgbm_seed_42.onnx
│       └── lgbm_seed_43.onnx
├── premade - данные, уже мной предобработанные
│   ├── msk-data-trend.csv
│   ├── occupancy_2025_popular_only.csv
│   ├── open-meteo-msk.csv
│   ├── route_flow_15min_clusters.csv
│   └── train_route_flow_15min_clusters.csv
├── report.ipynb - отчет
└── submission.csv - наш сабмит на Score = 0.88062
```
