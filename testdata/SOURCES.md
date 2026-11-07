# Test images

## Ultra HDR

`iphone-ultrahdr.jpg` is an iPhone 15 Pro Max photo in Display P3, CC0 1.0, taken
unchanged from Wikimedia Commons: [HK KTD Kwun Tong Apm mall shop Apple Store iPhone
display for sale Ipone 15 Pro Max May 2024 Ipone15 004.jpg](https://commons.wikimedia.org/wiki/File:HK_KTD_Kwun_Tong_Apm_mall_shop_Apple_Store_iPhone_display_for_sale_Ipone_15_Pro_Max_May_2024_Ipone15_004.jpg)
by KAiuphoupPor kou s s. SHA-1 `5726a10edf120195cd9b1cafbcee49ddb81a2081`, as
Commons lists it. It carries a gain map of Apple whose JPEG has no ICC profile,
which libultrahdr refuses to save unless the gain map is marked as sharing the
colour space of the image.
